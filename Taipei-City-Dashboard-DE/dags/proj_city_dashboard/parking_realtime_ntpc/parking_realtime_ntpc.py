from airflow import DAG
from operators.common_pipeline import CommonDag


def _parking_realtime_ntpc(**kwargs):
    import json
    import os

    import pandas as pd
    import requests
    from sqlalchemy import create_engine

    from utils.load_stage import (
        save_geodataframe_to_postgresql,
        update_lasttime_in_data_to_dataset_info,
    )
    from utils.transform_geometry import add_point_wkbgeometry_column_to_df
    from utils.transform_time import convert_str_to_time_format

    # Config
    dag_infos = kwargs.get("dag_infos")
    ready_data_db_uri = kwargs.get("ready_data_db_uri")
    dag_id = dag_infos.get("dag_id")
    load_behavior = dag_infos.get("load_behavior")
    default_table = dag_infos.get("ready_data_default_table")
    history_table = dag_infos.get("ready_data_history_table")
    URL = "https://apiatis.ntpc.gov.tw/ntpc-api/Parking/Parkinglot/NewTaipeiCity"
    GEOMETRY_TYPE = "Point"
    FROM_CRS = 4326

    engine = create_engine(ready_data_db_uri)

    # Extract
    res = requests.get(URL, timeout=60)
    res.raise_for_status()
    rows = res.json()
    if not isinstance(rows, list) or not rows:
        raise ValueError(f"Empty / unexpected payload from {URL}: {type(rows)}")
    raw_df = pd.DataFrame(rows)

    # Transform — NTPC keys are camelCase: id, area, name, summary, address,
    # tel, lon, lat, totalCar, totalMotor, availableCar, availableMotor,
    # updateTime. Some fields are null.
    df = raw_df.rename(columns={
        "id": "park_id",
        "lon": "lng",
        "totalCar": "car",
        "totalMotor": "motor",
        "availableCar": "available_car",
        "availableMotor": "available_motor",
        "updateTime": "data_time",
    })

    required = ["park_id", "name", "lng", "lat"]
    missing = [c for c in required if c not in df.columns]
    if missing:
        raise KeyError(f"Required columns missing after rename: {missing}; got {list(raw_df.columns)}")

    for c in ("car", "motor", "available_car", "available_motor"):
        df[c] = pd.to_numeric(df.get(c), errors="coerce").fillna(-1).astype(int)
    df["charging"] = 0  # NTPC feed does not expose charging spaces.

    df["lng"] = pd.to_numeric(df["lng"], errors="coerce")
    df["lat"] = pd.to_numeric(df["lat"], errors="coerce")
    df = df.dropna(subset=["lng", "lat"]).reset_index(drop=True)
    df = df[(df["lng"] > 119) & (df["lng"] < 123) & (df["lat"] > 21) & (df["lat"] < 26)].reset_index(drop=True)

    def _occupancy(row):
        total_car = row["car"] if row["car"] >= 0 else 0
        total_motor = row["motor"] if row["motor"] >= 0 else 0
        total = total_car + total_motor
        if total <= 0:
            return -99.0
        car_avail = row["available_car"] if row["available_car"] >= 0 else None
        motor_avail = row["available_motor"] if row["available_motor"] >= 0 else None
        if car_avail is None and motor_avail is None:
            return -99.0
        avail = (car_avail or 0) + (motor_avail or 0)
        return max(0.0, min(1.0, (total - avail) / total))

    df["occupied_rate"] = df.apply(_occupancy, axis=1)
    df["car"] = df["car"].clip(lower=0)
    df["motor"] = df["motor"].clip(lower=0)

    df["data_time"] = convert_str_to_time_format(df["data_time"])
    df["park_id"] = df["park_id"].astype(str)
    for c in ("area", "address", "summary", "tel"):
        if c not in df.columns:
            df[c] = None

    df = add_point_wkbgeometry_column_to_df(df, df["lng"], df["lat"], from_crs=FROM_CRS)

    ready_data = df[
        [
            "data_time",
            "park_id",
            "name",
            "area",
            "address",
            "summary",
            "tel",
            "car",
            "motor",
            "charging",
            "occupied_rate",
            "wkb_geometry",
        ]
    ].copy()

    # Load
    save_geodataframe_to_postgresql(
        engine,
        gdata=ready_data,
        load_behavior=load_behavior,
        default_table=default_table,
        history_table=history_table,
        geometry_type=GEOMETRY_TYPE,
    )
    update_lasttime_in_data_to_dataset_info(engine, dag_id, ready_data["data_time"].max())

    # B1 deviation: export GeoJSON to FE static dir.
    # See docs/road_speed_realtime_b1_deviation.md for context + cleanup plan.
    export_dir = "/opt/airflow/fe_mapdata"
    final_path = os.path.join(export_dir, "parking_realtime_ntpc.geojson")
    tmp_path = final_path + ".tmp"

    def _s(v):
        return v if (v is not None and pd.notna(v)) else None

    features = []
    for _, row in df.iterrows():
        features.append({
            "type": "Feature",
            "geometry": {"type": "Point", "coordinates": [float(row["lng"]), float(row["lat"])]},
            "properties": {
                "id": _s(row["park_id"]),
                "name": _s(row["name"]),
                "area": _s(row["area"]),
                "address": _s(row["address"]),
                "summary": _s(row["summary"]),
                "tel": _s(row["tel"]),
                "car": int(row["car"]),
                "motor": int(row["motor"]),
                "charging": int(row["charging"]),
                "occupied_rate": float(row["occupied_rate"]),
                "data_time": str(row["data_time"]) if pd.notna(row["data_time"]) else None,
            },
        })

    payload_out = {"type": "FeatureCollection", "features": features}
    with open(tmp_path, "w", encoding="utf-8") as fh:
        json.dump(payload_out, fh, ensure_ascii=False, separators=(",", ":"), allow_nan=False)
    os.replace(tmp_path, final_path)
    print(f"Exported {len(features)} features to {final_path}")


dag = CommonDag(proj_folder="proj_city_dashboard", dag_folder="parking_realtime_ntpc")
dag.create_dag(etl_func=_parking_realtime_ntpc)
