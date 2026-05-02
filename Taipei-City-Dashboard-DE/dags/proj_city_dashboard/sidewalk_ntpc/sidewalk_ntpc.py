from airflow import DAG
from operators.common_pipeline import CommonDag


def _sidewalk_ntpc(**kwargs):
    # Reuse the shared ETL body from sidewalk_tpe to avoid drift between the
    # two cities. The two DAGs differ only in source URL + ready_data table.
    from proj_city_dashboard.sidewalk_tpe.sidewalk_tpe import _run_sidewalk_etl
    from utils.transform_geometry import convert_geometry_to_wkbgeometry

    _run_sidewalk_etl(
        kwargs,
        url="https://openrouteservice.ydtw.net/ntpc.geojson",
        wkb_helper=convert_geometry_to_wkbgeometry,
    )


dag = CommonDag(proj_folder="proj_city_dashboard", dag_folder="sidewalk_ntpc")
dag.create_dag(etl_func=_sidewalk_ntpc)
