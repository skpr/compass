const http = require("http");

const DRUPAL_API = "http://127.0.0.1:8080/jsonapi/node/recipe";

function fetchRecipes() {
  return new Promise((resolve, reject) => {
    http.get(DRUPAL_API, (res) => {
      let data = "";
      res.on("data", (chunk) => (data += chunk));
      res.on("end", () => {
        try {
          resolve(JSON.parse(data));
        } catch (err) {
          reject(err);
        }
      });
      res.on("error", reject);
    }).on("error", reject);
  });
}

module.exports = { fetchRecipes };
