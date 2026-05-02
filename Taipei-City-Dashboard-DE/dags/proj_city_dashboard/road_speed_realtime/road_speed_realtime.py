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
