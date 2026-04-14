function stripHtml(html) {
  return html.replace(/<[^>]*>/g, "").trim();
}

function renderRecipes(recipes) {
  const items = recipes.data
    .map((recipe) => {
      const a = recipe.attributes;
      const summary = a.field_summary
        ? stripHtml(a.field_summary.processed)
        : "";
      return `
      <div style="border:1px solid #ddd;border-radius:8px;padding:16px;margin-bottom:16px;">
        <h2 style="margin:0 0 8px;">${a.title}</h2>
        <p style="color:#555;margin:0 0 12px;">${summary}</p>
        <table style="font-size:14px;">
          <tr><td style="padding-right:16px;"><strong>Difficulty</strong></td><td>${a.field_difficulty}</td></tr>
          <tr><td style="padding-right:16px;"><strong>Prep time</strong></td><td>${a.field_preparation_time} min</td></tr>
          <tr><td style="padding-right:16px;"><strong>Cooking time</strong></td><td>${a.field_cooking_time} min</td></tr>
          <tr><td style="padding-right:16px;"><strong>Servings</strong></td><td>${a.field_number_of_servings}</td></tr>
        </table>
      </div>`;
    })
    .join("\n");

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Recipes</title>
</head>
<body style="font-family:sans-serif;max-width:800px;margin:40px auto;padding:0 16px;">
  <h1>Recipes</h1>
  ${items}
</body>
</html>`;
}

module.exports = { stripHtml, renderRecipes };
