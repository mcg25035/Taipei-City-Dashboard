import { promises as fs } from "node:fs";
import path from "node:path";
import { findPedestrianRouteFeatureCollection } from "../utils/walkRouter.mjs";

const ORS_PROFILE = {
  car: "driving-car",
  biking: "cycling-regular",
};

const AVOID_SOURCE_PATH = "/geo_example.json";
const AVOID_RADIUS_M = 100;
const CIRCLE_SEGMENTS = 24;
const EARTH_RADIUS_M = 6_371_000;

function isLngLat(v) {
  return (
    Array.isArray(v) &&
    v.length >= 2 &&
    typeof v[0] === "number" &&
    typeof v[1] === "number" &&
    v[0] >= -180 &&
    v[0] <= 180 &&
    v[1] >= -90 &&
    v[1] <= 90
  );
}

function parseLngLat(raw) {
  if (!raw) return null;
  const parts = raw.split(",").map((s) => Number(s.trim()));
  if (parts.length !== 2 || parts.some((n) => !Number.isFinite(n))) return null;
  const candidate = [parts[0], parts[1]];
  return isLngLat(candidate) ? candidate : null;
}

function parseSearchParams(searchParams) {
  const origin = parseLngLat(searchParams.get("origin"));
  const destination = parseLngLat(searchParams.get("destination"));
  const mode = searchParams.get("mode");
  if (!origin) return { error: "origin must be 'lng,lat'" };
  if (!destination) return { error: "destination must be 'lng,lat'" };
  if (mode !== "car" && mode !== "biking" && mode !== "pedestrian") {
    return { error: "mode must be one of: car, biking, pedestrian" };
  }
  return { origin, destination, mode };
}

// Spherical destination point: build a polygon ring approximating a circle of
// radius_m metres around `center`. Returns a closed ring (last == first).
function bufferPoint(center, radius_m, segments = CIRCLE_SEGMENTS) {
  const [lng, lat] = center;
  const latRad = (lat * Math.PI) / 180;
  const lngRad = (lng * Math.PI) / 180;
  const angular = radius_m / EARTH_RADIUS_M;
  const ring = [];
  for (let i = 0; i < segments; i++) {
    const bearing = (2 * Math.PI * i) / segments;
    const sinLat =
      Math.sin(latRad) * Math.cos(angular) +
      Math.cos(latRad) * Math.sin(angular) * Math.cos(bearing);
    const newLat = Math.asin(sinLat);
    const newLng =
      lngRad +
      Math.atan2(
        Math.sin(bearing) * Math.sin(angular) * Math.cos(latRad),
        Math.cos(angular) - Math.sin(latRad) * sinLat,
      );
    ring.push([(newLng * 180) / Math.PI, (newLat * 180) / Math.PI]);
  }
  ring.push(ring[0]);
  return ring;
}

function isFeatureCollection(v) {
  if (!v || typeof v !== "object") return false;
  return v.type === "FeatureCollection" && Array.isArray(v.features);
}

// Recursively walk any geometry (including GeometryCollection) and call cb
// on every coordinate pair. Used to buffer every vertex into an avoidance circle.
function forEachVertex(geom, cb) {
  if (!geom || typeof geom !== "object") return;

  if (geom.type === "GeometryCollection" && Array.isArray(geom.geometries)) {
    for (const sub of geom.geometries) forEachVertex(sub, cb);
    return;
  }

  const visit = (coords) => {
    if (!Array.isArray(coords)) return;
    if (
      coords.length >= 2 &&
      typeof coords[0] === "number" &&
      typeof coords[1] === "number"
    ) {
      cb([coords[0], coords[1]]);
      return;
    }
    for (const c of coords) visit(c);
  };

  visit(geom.coordinates);
}

// Build a MultiPolygon avoidance geometry. For Polygon/MultiPolygon source
// features we keep the original geometry (so the interior is blocked) AND
// buffer the boundary; for Point/Line geometries we buffer every vertex.
function buildAvoidGeometry(fc) {
  const polys = [];
  let vertexCount = 0;

  for (const feature of fc.features) {
    const g = feature.geometry;
    if (!g || typeof g !== "object") continue;

    if (g.type === "Polygon" && Array.isArray(g.coordinates)) {
      polys.push(g.coordinates);
    } else if (g.type === "MultiPolygon" && Array.isArray(g.coordinates)) {
      for (const p of g.coordinates) polys.push(p);
    }

    forEachVertex(g, (p) => {
      vertexCount += 1;
      polys.push([bufferPoint(p, AVOID_RADIUS_M)]);
    });
  }

  return {
    geometry: polys.length
      ? { type: "MultiPolygon", coordinates: polys }
      : null,
    vertexCount,
    featureCount: fc.features.length,
  };
}

async function loadAvoidSource() {
  // The avoid source lives at public/geo_example.json. The previous Next.js
  // version fetched it over HTTP from its own /public origin; here we just
  // read it off the filesystem.
  const filePath = path.join(
    process.cwd(),
    "public",
    AVOID_SOURCE_PATH.replace(/^\/+/, ""),
  );
  let text;
  try {
    text = await fs.readFile(filePath, "utf8");
  } catch {
    return null;
  }
  if (!text.trim()) return null;
  let data;
  try {
    data = JSON.parse(text);
  } catch {
    return null;
  }
  return isFeatureCollection(data) ? data : null;
}

// ORS emits zero-length steps (notably the arrival step with
// way_points: [N, N]). Slicing those yields a single-point LineString, which
// is invalid GeoJSON. Pad backward by one so the step still represents the
// final approach segment.
function stepCoords(coords, [a, b]) {
  if (b > a) return coords.slice(a, b + 1);
  if (a > 0) return coords.slice(a - 1, b + 1);
  return [coords[a], coords[a]];
}

async function routeViaOrs(origin, destination, mode, avoid) {
  const apiKey = process.env.OPENROUTESERVICE_API_KEY;
  if (!apiKey) throw new Error("OPENROUTESERVICE_API_KEY is not set");

  const profile = ORS_PROFILE[mode];
  const url = `https://api.openrouteservice.org/v2/directions/${profile}/geojson`;

  const body = { coordinates: [origin, destination] };
  if (avoid) body.options = { avoid_polygons: avoid };

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: apiKey,
      "Content-Type": "application/json",
      Accept: "application/geo+json, application/json",
    },
    body: JSON.stringify(body),
  });

  const data = await res.json();
  if (!res.ok || !data.features?.length) {
    const msg =
      typeof data.error === "string"
        ? data.error
        : (data.error?.message ?? `HTTP ${res.status}`);
    throw new Error(`ORS: ${msg}`);
  }
  return data.features[0];
}

export async function handleNavigateAvoid(searchParams) {
  const parsed = parseSearchParams(searchParams);
  if ("error" in parsed) {
    return { status: 400, body: { error: parsed.error } };
  }

  const { origin, destination, mode } = parsed;

  try {
    const source = await loadAvoidSource();
    const avoid = source
      ? buildAvoidGeometry(source)
      : { geometry: null, vertexCount: 0, featureCount: 0 };

    if (mode === "pedestrian") {
      const walkFc = await findPedestrianRouteFeatureCollection(
        origin,
        destination,
        { avoid: avoid.geometry },
      );
      const fc = {
        type: "FeatureCollection",
        features: walkFc.features,
        metadata: {
          ...walkFc.metadata,
          avoid_source: AVOID_SOURCE_PATH,
          avoid_radius_m: AVOID_RADIUS_M,
          avoid_feature_count: avoid.featureCount,
          avoid_vertex_count: avoid.vertexCount,
        },
      };
      return { status: 200, body: fc };
    }

    const route = await routeViaOrs(origin, destination, mode, avoid.geometry);
    const coords = route.geometry.coordinates;
    const steps = route.properties.segments.flatMap((s) => s.steps);

    const fc = {
      type: "FeatureCollection",
      features: [
        {
          type: "Feature",
          geometry: route.geometry,
          properties: {
            mode,
            provider: "openrouteservice",
            distance_m: route.properties.summary.distance,
            duration_s: route.properties.summary.duration,
            avoid_source: AVOID_SOURCE_PATH,
            avoid_radius_m: AVOID_RADIUS_M,
            avoid_feature_count: avoid.featureCount,
            avoid_vertex_count: avoid.vertexCount,
          },
        },
        ...steps.map((s, i) => ({
          type: "Feature",
          geometry: {
            type: "LineString",
            coordinates: stepCoords(coords, s.way_points),
          },
          properties: {
            kind: "step",
            index: i,
            name: s.name,
            instruction: s.instruction,
            maneuver_type: s.type,
            distance_m: s.distance,
            duration_s: s.duration,
          },
        })),
      ],
    };

    return { status: 200, body: fc };
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Unknown routing error";
    return { status: 502, body: { error: message } };
  }
}
