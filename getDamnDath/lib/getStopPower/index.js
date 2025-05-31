/**
 * @typedef {Object} GeoJsonFeatureMetadata
 * @property {string} route_name 路名
 * @property {string|"null"} authority_name 權責單位名稱
 * @property {"台北市"|"新北市"} city 城市名稱
 * @property {string|"null"} town 鄉鎮區
 * @property {":D"} road_section_start 路段起點
 * @property {":D"} road_section_end 路段終點
 * @property {"雙向"} direction 行車方向
 * @property {"null"} cycling_type 自行車道型態
 * @property {267} cycling_length 自行車道長度
 * @property {"2013-12-31"} finished_time 完工時間
 * @property {"2025-03-14 00:02:31.000 +0800"} update_time 更新時間
 */

/**
 * @typedef {Object} GeoJsonFeature
 * @property {"Feature"} type GeoJSON Feature 類型
 * @property {GeoJsonFeatureMetadata} properties 屬性
 * @property {Object} geometry 幾何資訊
 * @property {"MultiLineStringd"} geometry.type 幾何類型
 * @property {[number, number][]} geometry.coordinates 座標
 */

/**
 * @typedef {Object} GeoJson
 * @property {"FeatureCollection"} type GeoJSON 類型
 * @property {string} name 隨機名稱
 * @property {Object} crs 坐標參考系統
 * @property {"name"} crs.type CRS 類型
 * @property {Object} crs.properties CRS 屬性
 * @property {"urn:ogc:def:crs:OGC:1.3:CRS84"} crs.properties.name CRS 名稱
 * @property {GeoJsonFeature[]} features GeoJSON Feature 陣列
 */

/**
 * @typedef {Object} packageJson
 * @property {"水"|"電"} type 類型
 * @property {number} timeMin 起始時間（timestamp）
 * @property {number} timeMax 結束時間（timestamp）
 * @property {GeoJson} data GeoJSON 資料
 */


import request from "request";
import * as cheerio from 'cheerio';
import getWayPosition from "../getWayPosition/index.js";
import fs from "fs";

const resourceUrl = "https://service.taipower.com.tw/branch/d102/xcnotice?xsmsid=0M242581312773778160";

function waterText2Date(text) {
  const dateRegex = /(\d{3}) 年 (\d{1,2}) 月 (\d{1,2}) 日/;

  const match = text.match(dateRegex);

  if (!match) {
    console.error("錯誤，無法解析斷水爬蟲日期:", text);
    return null;
  }

  const year = parseInt(match[1], 10) + 1911; // 民國轉西元
  const month = parseInt(match[2], 10);
  const day = parseInt(match[3], 10);

  const date = new Date(year, month - 1, day); // 月份從0開始計算

  if (isNaN(date.getTime())) {
    console.error("錯誤，無法解析斷水爬蟲日期:", text);
    return null;
  }

  return date;
}

function handleAddressText(text) {
  function cleanPart(part) {
    const chrTable = [
      "０",
      "１",
      "２",
      "３",
      "４",
      "５",
      "６",
      "７",
      "８",
      "９"
    ]

    for (let i = 0; i < chrTable.length; i++) {
      // part = part.replace(new , i.toString());
      part = part.replace(new RegExp(chrTable[i], "g"), i.toString());
    }

    const levelTable = ["市", "縣", "區", "鄉", "鎮", "市", "里", "鄰", "村", "路", "街", "巷", "弄", "段", "樓", "棟"];



    let rightestIndex = -1;
    for (let i = 0; i < levelTable.length; i++) {
      let isIncluded = false;
      for (let j = 0; j < levelTable[i].length; j++) {
        isIncluded = part.includes(levelTable[i][j]);
        if (isIncluded) {
          const index = part.indexOf(levelTable[i][j]);

          if (rightestIndex === -1 || index > rightestIndex) {
            rightestIndex = index;
          }
        }
      }
    }

    // debug 顯示完整原本的樣子，把要切掉的部分在終端機顯示紅色
    // let showPart = `${part.slice(0, rightestIndex + 1)}\x1b[31m${part.slice(rightestIndex+1)}\x1b[0m`;
    // console.log("處理地址:", showPart);

    //cut the part from the rightest index
    if (rightestIndex !== -1) {
      part = part.slice(0, rightestIndex + 1);
    }

    // console.log(rightestIndex, part);

    return part.trim();
  }

  if (text.endsWith("等")) {
    text = text.slice(0, -1).trim();
  }

  const parts = text.split("，")
    .map(part => cleanPart(part));

  // remove same parts
  const uniqueParts = new Set(parts);
  // convert back to array
  parts.length = 0; // clear the original array

  let GeoJsonFeatures = [];

  uniqueParts.forEach(part => {
    const string = "";
    const number = 0;

    let GeoJsonFeatureMetadata = {
      route_name: part,
      authority_name: "null",
      city: part.includes("市") ? part.split("市")[0] + "市" : "null",
      town: "null",
      // road_section_start: "D:",
      // road_section_end: ":D",
      road_section_start: "和平西路二段104巷",
      road_section_end: "和平西路二段98巷",
      direction: "雙向",
      cycling_type: "null",
      cycling_length: 267,
      finished_time: "2013-12-31",
      update_time: "2025-03-14 00:02:31.000 +0800",
    }

    let GeoJsonFeature = {
      type: "Feature",
      properties: GeoJsonFeatureMetadata,
      geometry: {
        type: "MultiLineString",
        coordinates: [0, 0],
      }
    }

    GeoJsonFeatures.push(GeoJsonFeature);
  });
  return GeoJsonFeatures;
}

let id = 100000;
function genId() {
  function encode(n) {
    let str = ((n + 1) * 2124).toString(36);
    const shift = (n * 97) % 36;
    const pattern = "0123456789abcdefghijklmnopqrstuvwxyz";
    let ret = "";
    for (let i = 0; i < str.length; i++) {
      ret += pattern[(pattern.indexOf(str[i]) + shift) % pattern.length];
    }
    ret = pattern[shift] + ret;
    return ret.toUpperCase();
  }

  id++;
  return encode(id);
}

async function getStopPowerWays() {
  return new Promise((resolve, reject) => {
    request({
      url: resourceUrl,
      method: "GET"
    }, (error, res, body) => {
      if (error) {
        console.error("錯誤，在取得水資料時發生錯誤:", error);
        reject(new Error(`在取得水資料時發生錯誤: ${error.message}`));
      }
      if (res.statusCode !== 200) {
        console.error("錯誤，無法取得水資料，狀態碼:", res.statusCode);
        reject(new Error(`無法取得水資料，狀態碼: ${res.statusCode}`));
      }

      if (!body) {
        console.error("錯誤，沒有取得水資料的內容");
        reject(new Error("沒有取得水資料的內容"));
      }

      const $ = cheerio.load(body);
      const stopWaterDays = $(".ListTable");

      if (stopWaterDays.length === 0) {
        if (body.includes("目前無停水資訊")) {
          console.log("目前無停水資訊");
          resolve([]);
        }
      }

      const packages = [];

      stopWaterDays.each((index, element) => {
        const string_ = "";
        const number_ = 0;

        const trs = $(element).find("tr");
        const dateText = $(element).find("caption").text().trim();
        const date = waterText2Date(dateText);

        // remove the first row 
        trs.splice(0, 1);

        // trs.each((i, tr) => {
        for (const tr of trs) {
          let geoJson = {
            type: "FeatureCollection",
            name: string_,
            crs: {
              type: "name",
              properties: {
                name: "urn:ogc:def:crs:OGC:1.3:CRS84",
              },
            },
            features: undefined,
          }

          let packageJson = {
            type: "電",
            timeMin: number_,
            timeMax: number_,
            data: geoJson,
          }

          const tds = $(tr).find("td");
          const time = $(tds[0]).text();
          // '自 0 時 0 分至 3 時 0 分' use regex to extract time
          const timeRegex = /自 (\d{1,2}) 時 (\d{1,2}) 分至 (\d{1,2}) 時 (\d{1,2}) 分/;
          const timeMatch = time.match(timeRegex);

          const area = $(tds[1]).contents().last().text().trim();

          if (!date || !timeMatch || !area) {
            console.warn("警告，無法解析停水資料，可能是格式不正確", {
              dateText,
              time,
              area
            });
            reject(new Error("無法解析停水資料，可能是格式不正確"));
            return;
          }

          packageJson.timeMin = new Date(date.getFullYear(), date.getMonth(), date.getDate(), parseInt(timeMatch[1]), parseInt(timeMatch[2]));
          packageJson.timeMax = new Date(date.getFullYear(), date.getMonth(), date.getDate(), parseInt(timeMatch[3]), parseInt(timeMatch[4]));

          geoJson.name = `object-${genId()}`;
          geoJson.features = handleAddressText(area)


          packages.push(packageJson);
          // });
        }
      });

      if (packages.length === 0) {
        console.error("錯誤，沒有取得停水資料");
        reject(new Error("沒有取得停水資料"));
      } else {
        resolve(packages);
      }
    });
  });
}


async function getStopPower() {
  try {
    /**
     * @type {packageJson[]}
     */
    const packages = await getStopPowerWays();

    // console.log(packages[0].data.features.length)

    // return;

    let addressCount = 0;
    for (let i = 0; i < packages.length; i++) {
      const packageJson = packages[i];
      addressCount += packageJson.data.features.length;
      // console.log(packageJson.data.features.length);
    }

    console.log(`共 ${packages.length} 個停電資料包，每個包包含 ${addressCount} 個地址`);

    // debug
    // return;


    // console.log(`Total addresses: ${addressCount}`);

    let count = 0;

    // progress bar
    const startTime = Date.now();
    // const Running = setInterval(() => {
    // const bar = `[${'='.repeat(filledLength > 0 ? filledLength - 1 : 0)}${filledLength > 0 ? '>' : ''}${' '.repeat(barLength - filledLength)}] ${percent}%`;
    //   // console.log(`\r${bar} ${count}/${addressCount} addresses`);
    //   process.stdout.clearLine();
    //   process.stdout.cursorTo(0);
    //   process.stdout.write(`${bar} ${count}/${addressCount} addresses, use ${((Date.now() - startTime) / 1000).toFixed(2)}s, ${((count / (Date.now() - startTime)) * 1000).toFixed(2)} addresses/s`);
    // }, 200);

    let status = "➡️"
    for (let i = 0; i < packages.length; i++) {
      const packageJson = packages[i];
      for (let i = 0; i < packageJson.data.features.length; i++) {
        const address = packageJson.data.features[i].properties.route_name;


        const percent = addressCount === 0 ? 0 : Math.floor((count / addressCount) * 100);
        const barLength = 60;
        const filledLength = Math.floor((percent / 100) * barLength);
        let bar = `[${'='.repeat(filledLength > 0 ? filledLength - 1 : 0)}${filledLength > 0 ? '>' : ''}${' '.repeat(barLength - filledLength)}] ${percent}%`;
        // process.stdout.clearLine();
        process.stdout.cursorTo(0, 0);
        // 預計剩餘時間
        const elapsed = (Date.now() - startTime) / 1000;
        const speed = count > 0 ? elapsed / count : 0;
        const remaining = addressCount - count;
        const eta = speed > 0 ? (remaining * speed) : 0;
        process.stdout.write(`last：${status} now：${address.padEnd(60, ' ')}\n${bar} [${i}/${packageJson.data.features.length}] ${count}/${addressCount} , use ${elapsed.toFixed(2)}s, ${((count / (Date.now() - startTime)) * 1000).toFixed(2)} req/s, 預計剩餘時間: ${eta.toFixed(2)}s `);


        const position = await getWayPosition(address);
        packageJson.data.features[i].geometry.coordinates = [position];


        count++;
        // bar = `[${'='.repeat(filledLength > 0 ? filledLength - 1 : 0)}${filledLength > 0 ? '>' : ''}${' '.repeat(barLength - filledLength)}] ${percent}%`;
        // process.stdout.clearLine();
        // process.stdout.cursorTo(0, 0);
        status = position.length == 0 ? "🔴" : "✅";
        // process.stdout.write(`${status} ${address.padEnd(80, ' ')}\n${bar} ${count}/${addressCount} addresses, use ${((Date.now() - startTime) / 1000).toFixed(2)}s, ${((count / (Date.now() - startTime)) * 1000).toFixed(2)} addresses/s`);

        // await fs.writeFileSync('stop_water_data.json', JSON.stringify(packages), 'utf8');
      }


      const CBURL = "http://23.146.248.105:8080/api/v1/infraDown"
      const options = {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          type: packageJson.type,
          timeMin: packageJson.timeMin.getTime(),
          timeMax: packageJson.timeMax.getTime(),
          data: JSON.stringify(packageJson.data)
        })
      };
      process.stdout.clearLine();
      process.stdout.cursorTo(0, 0);
      process.stdout.write(`\n\n➡️ 正在上傳停電資料包 ${i + 1}/${packages.length}... `);
      try {
        const response = await fetch(CBURL, options);
        // 將 response 寫到檔案
        const responseText = await response.text();
        fs.writeFileSync(`stop_power_response_${i + 1}.json`, JSON.stringify({
          requestBody: JSON.parse(options.body),
          response: responseText
        }, null, 2), 'utf8');
        if (JSON.parse(responseText).status == "error") {
          // process.stdout.write(`\n\n成功，上傳停水資料包 ${i + 1}/${packages.length}`);
          process.stdout.clearLine();
          process.stdout.cursorTo(0, 0);
          process.stdout.write(`\n\n🔴 錯誤，上傳停電資料包 ${i + 1}/${packages.length} 時發生錯誤，狀態碼: ${response.status}`);
        } else {
          // console.log(`成功，上傳停水資料包 ${i + 1}/${packages.length}`);
          process.stdout.clearLine();
          process.stdout.cursorTo(0, 0);
          process.stdout.write(`\n\n✅ 成功，上傳停電資料包 ${i + 1}/${packages.length}`);
        }
      } catch (error) {
        console.error("錯誤，上傳停水資料時發生錯誤:", error);
        // process.stdout.write(`\n\n➡️ 正在上傳停電資料包 ${i + 1}/${packages.length}... `);
        process.stdout.write(`\n\n🔴 錯誤，上傳停電資料包 ${i + 1}/${packages.length} 時發生錯誤: ${error.message}`);
      }
    }

    const percent = addressCount === 0 ? 0 : Math.floor((count / addressCount) * 100);
    const barLength = 60;
    const filledLength = Math.floor((percent / 100) * barLength);
    bar = `[${'='.repeat(filledLength > 0 ? filledLength - 1 : 0)}${filledLength > 0 ? '>' : ''}${' '.repeat(barLength - filledLength)}] ${percent}%`;
    process.stdout.clearLine();
    process.stdout.cursorTo(0, 0);
    status = position.length == 0 ? "🔴" : "✅";
    process.stdout.write(`last：${status} now：${address.padEnd(80, ' ')}\n${bar} ${count}/${addressCount} addresses, use ${((Date.now() - startTime) / 1000).toFixed(2)}s, ${((count / (Date.now() - startTime)) * 1000).toFixed(2)} addresses/s`);

    // clearInterval(Running);
    console.log("All addresses processed");

    // console.log(result);

    // Save the result to a JSON file
    // const fs = require('fs');
    fs.writeFileSync('stop_water_data.json', JSON.stringify(packages, null, 2), 'utf8');
    console.log("Stop water data saved to stop_water_data.json");
  } catch (error) {
    console.error("Error fetching stop water data:", error);
  }
}



export default getStopPower;



getStopPower();