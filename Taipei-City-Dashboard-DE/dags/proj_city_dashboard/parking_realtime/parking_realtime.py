from airflow import DAG
from operators.common_pipeline import CommonDag


def _parking_realtime(**kwargs):
    import json
    import os

    import pandas as pd
    import requests
    from pyproj import Transformer
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
    DESC_URL = "https://tcgbusfs.blob.core.windows.net/blobtcmsv/TCMSV_alldesc.json"
    AVAIL_URL = "https://tcgbusfs.blob.core.windows.net/blobtcmsv/TCMSV_allavailable.json"
    GEOMETRY_TYPE = "Point"
    TARGET_CRS = 4326

    engine = create_engine(ready_data_db_uri)

    # Extract — descriptions (static-ish) + realtime availability, joined on id
    desc_res = requests.get(DESC_URL, timeout=60)
    desc_res.raise_for_status()
    desc_payload = desc_res.json()
    desc_rows = desc_payload.get("data", {}).get("park", [])
    if not desc_rows:
        raise ValueError(f"Empty desc payload from {DESC_URL}")

    avail_res = requests.get(AVAIL_URL, timeout=60)
    avail_res.raise_for_status()
    avail_payload = avail_res.json()
    avail_rows = avail_payload.get("data", {}).get("park", [])
    avail_update_time = avail_payload.get("data", {}).get("UPDATETIME")

    desc_df = pd.DataFrame(desc_rows)
    avail_df = pd.DataFrame(avail_rows)

    # Transform — coerce numeric fields, drop charging-station-only rows.
    for c in ("totalcar", "totalmotor"):
        if c not in desc_df.columns:
            desc_df[c] = 0
        desc_df[c] = pd.to_numeric(desc_df[c], errors="coerce").fillna(0).astype(int)
    if "ChargingStation" in desc_df.columns:
        desc_df["charging"] = pd.to_numeric(desc_df["ChargingStation"], errors="coerce").fillna(0).astype(int)
    else:
        desc_df["charging"] = 0

    if "availablecar" not in avail_df.columns:
        avail_df["availablecar"] = -9
    avail_df["availablecar"] = pd.to_numeric(avail_df["availablecar"], errors="coerce").fillna(-9).astype(int)
    if "availablemotor" not in avail_df.columns:
        avail_df["availablemotor"] = -9
    avail_df["availablemotor"] = pd.to_numeric(avail_df["availablemotor"], errors="coerce").fillna(-9).astype(int)

    df = desc_df.merge(
        avail_df[["id", "availablecar", "availablemotor"]],
        on="id",
        how="left",
    )
    df["availablecar"] = df["availablecar"].fillna(-9).astype(int)
    df["availablemotor"] = df["availablemotor"].fillna(-9).astype(int)

    # Geometry: TWD97 (EPSG:3826) tw97x/tw97y → WGS84. Drop rows w/o coords.
    df["tw97x"] = pd.to_numeric(df.get("tw97x"), errors="coerce")
    df["tw97y"] = pd.to_numeric(df.get("tw97y"), errors="coerce")
    df = df.dropna(subset=["tw97x", "tw97y"]).reset_index(drop=True)
    df = df[(df["tw97x"] != 0) & (df["tw97y"] != 0)].reset_index(drop=True)

    transformer = Transformer.from_crs("EPSG:3826", "EPSG:4326", always_xy=True)
    lng_arr, lat_arr = transformer.transform(df["tw97x"].values, df["tw97y"].values)
    df["lng"] = lng_arr
    df["lat"] = lat_arr
    # Sanity: drop anything outside Taiwan bbox.
    df = df[(df["lng"] > 119) & (df["lng"] < 123) & (df["lat"] > 21) & (df["lat"] < 26)].reset_index(drop=True)

    # occupied_rate: -99 if no realtime data, else (total - available) / total.
    def _occupancy(row):
        total = (row["totalcar"] or 0) + (row["totalmotor"] or 0)
        if total <= 0:
            return -99.0
        car_avail = row["availablecar"] if row["availablecar"] >= 0 else None
        motor_avail = row["availablemotor"] if row["availablemotor"] >= 0 else None
        if car_avail is None and motor_avail is None:
            return -99.0
        avail = (car_avail or 0) + (motor_avail or 0)
        return max(0.0, min(1.0, (total - avail) / total))

    df["occupied_rate"] = df.apply(_occupancy, axis=1)

    df["data_time"] = convert_str_to_time_format(pd.Series([avail_update_time] * len(df)))
    df["park_id"] = df["id"].astype(str)
    df["car"] = df["totalcar"].astype(int)
    df["motor"] = df["totalmotor"].astype(int)
    for c in ("area", "address", "summary", "tel"):
        if c not in df.columns:
            df[c] = None

    df = add_point_wkbgeometry_column_to_df(df, df["lng"], df["lat"], from_crs=TARGET_CRS)

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
    final_path = os.path.join(export_dir, "parking_realtime.geojson")
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


dag = CommonDag(proj_folder="proj_city_dashboard", dag_folder="parking_realtime")
dag.create_dag(etl_func=_parking_realtime)
