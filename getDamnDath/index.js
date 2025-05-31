import { count } from "console";
import getStopPower from "./lib/getStopPower/index.js";
import fs from 'fs';

async function main() {
  const stopPower = await getStopPower();
  const outputFile = 'stopPower.json';

  fs.writeFile(outputFile, JSON.stringify(stopPower, null, 2), (err) => {
    if (err) {
      console.error("錯誤，無法寫入文件:", err);
    } else {
      console.log(`成功，已將停靠站功率數據寫入 ${outputFile}`);
    }
  });
}



main();