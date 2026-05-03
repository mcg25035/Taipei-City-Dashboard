// TWD97 TM2 (EPSG:3826) <-> WGS84 lng/lat conversion.
//
// Replaces proj4 for the single projection we need. Implements the standard
// inverse Transverse Mercator formulas (Snyder, "Map Projections — A Working
// Manual", §8) on the GRS80 ellipsoid. WGS84 and GRS80 differ by ~0.1 mm at
// the surface, so we treat them as the same datum (matching what proj4 does
// when no explicit datum shift is given).
//
// Projection parameters (matching the proj4 string used previously):
//   +proj=tmerc +lat_0=0 +lon_0=121 +k=0.9999 +x_0=250000 +y_0=0 +ellps=GRS80

const A = 6378137; // GRS80 semi-major axis (m)
const F = 1 / 298.257222101; // GRS80 flattening
const E2 = 2 * F - F * F; // first eccentricity squared
const EP2 = E2 / (1 - E2); // second eccentricity squared

const LON0 = (121 * Math.PI) / 180;
const K0 = 0.9999;
const FE = 250000;
const FN = 0;

// Pre-computed constant for footprint latitude series.
const E1 = (1 - Math.sqrt(1 - E2)) / (1 + Math.sqrt(1 - E2));

// TWD97 TM2 (E, N in metres) -> WGS84 [lng, lat] in degrees.
export function twd97ToWgs84([x, y]) {
  const M = (y - FN) / K0;
  const mu =
    M / (A * (1 - E2 / 4 - (3 * E2 * E2) / 64 - (5 * E2 * E2 * E2) / 256));

  const e1_2 = E1 * E1;
  const e1_3 = e1_2 * E1;
  const e1_4 = e1_3 * E1;

  const phi1 =
    mu +
    ((3 * E1) / 2 - (27 * e1_3) / 32) * Math.sin(2 * mu) +
    ((21 * e1_2) / 16 - (55 * e1_4) / 32) * Math.sin(4 * mu) +
    ((151 * e1_3) / 96) * Math.sin(6 * mu) +
    ((1097 * e1_4) / 512) * Math.sin(8 * mu);

  const sinPhi1 = Math.sin(phi1);
  const cosPhi1 = Math.cos(phi1);
  const tanPhi1 = Math.tan(phi1);

  const C1 = EP2 * cosPhi1 * cosPhi1;
  const T1 = tanPhi1 * tanPhi1;
  const N1 = A / Math.sqrt(1 - E2 * sinPhi1 * sinPhi1);
  const R1 = (A * (1 - E2)) / Math.pow(1 - E2 * sinPhi1 * sinPhi1, 1.5);
  const D = (x - FE) / (N1 * K0);

  const D2 = D * D;
  const D3 = D2 * D;
  const D4 = D3 * D;
  const D5 = D4 * D;
  const D6 = D5 * D;

  const phi =
    phi1 -
    ((N1 * tanPhi1) / R1) *
      (D2 / 2 -
        ((5 + 3 * T1 + 10 * C1 - 4 * C1 * C1 - 9 * EP2) * D4) / 24 +
        ((61 +
          90 * T1 +
          298 * C1 +
          45 * T1 * T1 -
          252 * EP2 -
          3 * C1 * C1) *
          D6) /
          720);

  const lambda =
    LON0 +
    (D -
      ((1 + 2 * T1 + C1) * D3) / 6 +
      ((5 - 2 * C1 + 28 * T1 - 3 * C1 * C1 + 8 * EP2 + 24 * T1 * T1) * D5) /
        120) /
      cosPhi1;

  return [(lambda * 180) / Math.PI, (phi * 180) / Math.PI];
}
