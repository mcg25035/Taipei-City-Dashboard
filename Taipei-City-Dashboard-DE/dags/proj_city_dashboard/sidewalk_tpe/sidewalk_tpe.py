from airflow import DAG
from operators.common_pipeline import CommonDag


def _sidewalk_tpe(**kwargs):
    from utils.transform_geometry import convert_geometry_to_wkbgeometry
    _run_sidewalk_etl(
        kwargs,
        url="https://openrouteservice.ydtw.net/taipei.geojson",
        wkb_helper=convert_geometry_to_wkbgeometry,
    )


def _run_sidewalk_etl(kwargs, url, wkb_helper):
    """Shared ETL body. Mirrors road_obstacle.py shape but for OSM sidewalk lines."""
    import geopandas as gpd
    import pandas as pd
    import requests
    from shapely.geometry import shape
    from sqlalchemy import create_engine

    from utils.load_stage import (
        save_geodataframe_to_postgresql,
        update_lasttime_in_data_to_dataset_info,
    )

    dag_infos = kwargs.get("dag_infos")
    ready_data_db_uri = kwargs.get("ready_data_db_uri")
    dag_id = dag_infos.get("dag_id")
    load_behavior = dag_infos.get("load_behavior")
    default_table = dag_infos.get("ready_data_default_table")
    history_table = dag_infos.get("ready_data_history_table")
    GEOMETRY_TYPE = "LineString"
    FROM_CRS = 4326

    engine = create_engine(ready_data_db_uri)

    res = requests.get(url, timeout=120)
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

    # Rename OSM colon-tags to snake_case Postgres columns. Drop the long tail
    # of OSM keys we don't surface in the FE.
    rename_map = {
        "@id": "osm_id",
        "name:en": "name_en",
        "name:zh": "name_zh",
    }
    gdata = raw.rename(columns=rename_map)
    gdata["data_time"] = pd.Timestamp.utcnow()
    gdata = wkb_helper(gdata, from_crs=FROM_CRS)

    keep_columns = [
        "data_time",
        "osm_id",
        "name",
        "name_en",
        "name_zh",
        "highway",
        "footway",
        "surface",
        "oneway",
        "width",
        "wheelchair",
        "tactile_paving",
        "wkb_geometry",
    ]
    ready_data = gdata.reindex(columns=keep_columns)

    save_geodataframe_to_postgresql(
        engine,
        gdata=ready_data,
        load_behavior=load_behavior,
        default_table=default_table,
        history_table=history_table,
        geometry_type=GEOMETRY_TYPE,
    )
    update_lasttime_in_data_to_dataset_info(engine, dag_id, ready_data["data_time"].max())


dag = CommonDag(proj_folder="proj_city_dashboard", dag_folder="sidewalk_tpe")
dag.create_dag(etl_func=_sidewalk_tpe)
