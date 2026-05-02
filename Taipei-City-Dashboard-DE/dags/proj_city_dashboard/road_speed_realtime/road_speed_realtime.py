from airflow import DAG
from operators.common_pipeline import CommonDag


def _road_speed_realtime(**kwargs):
    import geopandas as gpd
    import pandas as pd
    import requests
    from shapely.geometry import shape
    from sqlalchemy import create_engine

    from utils.load_stage import (
        save_geodataframe_to_postgresql,
        update_lasttime_in_data_to_dataset_info,
    )
    from utils.transform_geometry import convert_geometry_to_wkbgeometry
    from utils.transform_time import convert_str_to_time_format

    # Config
    dag_infos = kwargs.get("dag_infos")
    ready_data_db_uri = kwargs.get("ready_data_db_uri")
    dag_id = dag_infos.get("dag_id")
    load_behavior = dag_infos.get("load_behavior")
    default_table = dag_infos.get("ready_data_default_table")
    history_table = dag_infos.get("ready_data_history_table")
    URL = "https://itsapi.taipei.gov.tw/TPTS_WEB_API/roadInformation/roadSpeedGeoJSON"
    GEOMETRY_TYPE = "Polygon"
    FROM_CRS = 4326

    engine = create_engine(ready_data_db_uri)

    # Extract — POST returns GeoJSON FeatureCollection
    res = requests.post(URL, timeout=30)
    res.raise_for_status()
    fc = res.json()
    features = fc.get("features", [])
    if not features:
        raise ValueError("Empty FeatureCollection from source.")

    rows = []
    geometries = []
    for f in features:
        p = f.get("properties", {})
        rows.append(p)
        geometries.append(shape(f["geometry"]) if f.get("geometry") else None)
    raw_data = gpd.GeoDataFrame(rows, geometry=geometries, crs=f"EPSG:{FROM_CRS}")

    # Transform
    gdata = raw_data.rename(
        columns={
            "sectionId": "section_id",
            "sectionName": "section_name",
            "travelSpeed": "travel_speed",
            "dataCollectTime": "data_time",
            "levelName": "level_name",
            "levelColor": "level_color",
        }
    )
    gdata["travel_speed"] = pd.to_numeric(gdata["travel_speed"], errors="coerce")
    gdata["data_time"] = convert_str_to_time_format(gdata["data_time"])
    gdata = convert_geometry_to_wkbgeometry(gdata, from_crs=FROM_CRS)

    ready_data = gdata[
        [
            "section_id",
            "section_name",
            "travel_speed",
            "level_name",
            "level_color",
            "data_time",
            "wkb_geometry",
        ]
    ]

    # Load
    save_geodataframe_to_postgresql(
        engine,
        gdata=ready_data,
        load_behavior=load_behavior,
        default_table=default_table,
        history_table=history_table,
        geometry_type=GEOMETRY_TYPE,
    )
    lasttime_in_data = ready_data["data_time"].max()
    update_lasttime_in_data_to_dataset_info(engine, dag_id, lasttime_in_data)

    # ------------------------------------------------------------------
    # B1 deviation: export GeoJSON to FE static dir.
    # See docs/road_speed_realtime_b1_deviation.md for context + cleanup plan.
    # ------------------------------------------------------------------
    import json
    import os
    from shapely.geometry import mapping

    export_dir = "/opt/airflow/fe_mapdata"
    final_path = os.path.join(export_dir, "traffic_road_speed_realtime.geojson")
    tmp_path = final_path + ".tmp"

    # Reproject back to EPSG:4326 for the static GeoJSON consumed by Mapbox.
    export_gdf = gdata.to_crs(epsg=4326)
    export_features = []
    for _, row in export_gdf.iterrows():
        geom = row.get("geometry")
        if geom is None:
            continue
        export_features.append({
            "type": "Feature",
            "geometry": mapping(geom),
            "properties": {
                "section_id": row["section_id"],
                "section_name": row["section_name"],
                "travel_speed": float(row["travel_speed"]) if pd.notna(row["travel_speed"]) else None,
                "level_name": row["level_name"],
                "level_color": row["level_color"],
                "data_time": str(row["data_time"]) if pd.notna(row["data_time"]) else None,
            },
        })

    payload = {"type": "FeatureCollection", "features": export_features}
    with open(tmp_path, "w", encoding="utf-8") as fh:
        json.dump(payload, fh, ensure_ascii=False, separators=(",", ":"))
    os.replace(tmp_path, final_path)
    print(f"Exported {len(export_features)} features to {final_path}")

    # Mirror for road_travel_speed_realtime component (heatmap on travel_speed).
    # Same payload, distinct map_config.index → distinct static file path.
    mirror_path = os.path.join(export_dir, "traffic_road_travel_speed_realtime.geojson")
    mirror_tmp = mirror_path + ".tmp"
    with open(mirror_tmp, "w", encoding="utf-8") as fh:
        json.dump(payload, fh, ensure_ascii=False, separators=(",", ":"))
    os.replace(mirror_tmp, mirror_path)
    print(f"Mirrored {len(export_features)} features to {mirror_path}")


dag = CommonDag(proj_folder="proj_city_dashboard", dag_folder="road_speed_realtime")
dag.create_dag(etl_func=_road_speed_realtime)
