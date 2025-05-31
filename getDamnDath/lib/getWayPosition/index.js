const API_BASE_URL = "https://overpass-api.de/api/interpreter";
const OVERPASS_QUERY = (wayName) => `
[out:json][timeout:25];
area["name:zh"="${wayName.split("區")[0].split("市")[1]}區"]["boundary"="administrative"]->.searchArea;
(
  way(area.searchArea)["name"~"${wayName.split("區")[1]}"];
);
out body;
>;
out skel qt;
`;

async function getWayPosition(wayName) {
  return new Promise(async (resolve, reject) => {
    const options = {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded"
      },
      body: OVERPASS_QUERY(wayName)
    };

    // try {
      const response = await fetch(API_BASE_URL, options);
      if (!response.ok) {
        console.error("錯誤，無法獲取道路位置，狀態碼:", response.status);
        reject(new Error(`無法獲取道路位置，狀態碼: ${response.status}`));
      }
      const data = await response.json();

      if (!data.elements || data.elements.length === 0) {
        // console.warn("警告，找不到任何元素，可能是道路名稱不正確或不存在");
        resolve([]);
        return;
      }

      // console.log(data.elements.length == 0);

      const points = [];
      // if (data.elements.length === 0) {
      //   console.log("警告，沒有找到任何元素");
      //   resolve([]);
      //   return;        
      // }
      // console.log(":D")


      data.elements[0].nodes.forEach(nodeId => {
        const node = data.elements.find(el => el.type === "node" && el.id === nodeId);
        if (node) {
          points.push([
            node.lat,
            node.lon
          ]);
        }
      });

      resolve(points);
    // } catch (error) {
    //   console.error("錯誤，獲取道路位置時發生錯誤:", error.message);
    //   reject(new Error(`獲取道路位置時發生錯誤: ${error.message}`));
    // }
  });
}

export default getWayPosition;