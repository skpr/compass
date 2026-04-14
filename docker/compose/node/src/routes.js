const express = require("express");
const { fetchRecipes } = require("./drupal");
const { renderRecipes } = require("./render");

const SLOW_DELAY_MS = 5000;

const router = express.Router();

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

router.get("/", async (req, res) => {
  try {
    const recipes = await fetchRecipes();
    const html = renderRecipes(recipes);
    res.type("html").send(html);
  } catch (err) {
    res.status(502).type("text").send("Failed to fetch recipes from Drupal: " + err.message);
  }
});

router.get("/slow", async (req, res) => {
  await delay(SLOW_DELAY_MS);
  res.type("html").send(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Slow Response</title>
</head>
<body style="font-family:sans-serif;max-width:800px;margin:40px auto;padding:0 16px;">
  <h1>Slow Response</h1>
  <p>This response was delayed by ${SLOW_DELAY_MS / 1000} seconds.</p>
</body>
</html>`);
});

module.exports = router;
