from airflow import DAG
from operators.common_pipeline import CommonDag


def _road_obstacle(**kwargs):
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

    # Config
    dag_infos = kwargs.get("dag_infos")
    ready_data_db_uri = kwargs.get("ready_data_db_uri")
    dag_id = dag_infos.get("dag_id")
    load_behavior = dag_infos.get("load_behavior")
    default_table = dag_infos.get("ready_data_default_table")
    history_table = dag_infos.get("ready_data_history_table")
    URL = "https://openrouteservice.ydtw.net/geo_example.json"
    GEOMETRY_TYPE = "Point"
    FROM_CRS = 4326

    engine = create_engine(ready_data_db_uri)

    # Extract
    res = requests.get(URL, timeout=30)
    res.raise_for_status()
    fc = res.json()
    features = fc.get("features", [])
    if not features:
        raise ValueError("Empty FeatureCollection from source.")

    rows = []
    geoms = []
    for f in features:
        rows.append(f.get("properties", {}) or {})
        geoms.append(shape(f["geometry"]) if f.get("geometry") else None)
    raw = gpd.GeoDataFrame(rows, geometry=geoms, crs=f"EPSG:{FROM_CRS}")

    # Transform — properties already lowercase + snake_case-ish
    gdata = raw.copy()
    gdata["data_time"] = pd.Timestamp.utcnow()
    gdata = convert_geometry_to_wkbgeometry(gdata, from_crs=FROM_CRS)

    ready_data = gdata[
        [
            "data_time",
            "city",
            "district",
            "road",
            "house",
            "address",
            "address2",
            "apply_type",
            "provider",
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
    update_lasttime_in_data_to_dataset_info(engine, dag_id, ready_data["data_time"].max())


dag = CommonDag(proj_folder="proj_city_dashboard", dag_folder="road_obstacle")
dag.create_dag(etl_func=_road_obstacle)
