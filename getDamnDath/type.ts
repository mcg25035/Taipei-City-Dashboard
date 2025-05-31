interface GeoJsonFeatureMetadata {
  route_name: string; // 路名
  authority_name: string | null;
  city: "台北市" | "新北市";
  town: string | null;
  road_section_start: ":D";
  road_section_end: ":D";
  direction: "雙向";
  cycling_type: "null";
  cycling_length: 267;
  finished_time: "2013-12-31";
  update_time: "2025-03-14 00:02:31.000 +0800";
}

interface GeoJsonFeature {
  type: "Feature";
  properties: GeoJsonFeatureMetadata;
  geometry: {
    type: "MultiLineStringd";
    coordinates: [number, number][];
  };
}

interface GeoJson {
  type: "FeatureCollection";
  name: string; // random name
  crs: {
    type: "name";
    properties: {
      name: "urn:ogc:def:crs:OGC:1.3:CRS84";
    };
  };
  features: GeoJsonFeature[];
}

interface packageJson {
  type: "水" | "電";
  timeMin: number; // v
  timeMax: number; // v
  data: GeoJson;
}
