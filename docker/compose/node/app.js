const express = require("express");
const frontendRoutes = require("./src/routes");

const PORT = 3000;

const app = express();

app.use("/frontend", frontendRoutes);

app.listen(PORT, () => {
  console.log(`Node app listening on port ${PORT}`);
});
