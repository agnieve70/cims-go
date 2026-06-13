import fs from "node:fs/promises";
import path from "node:path";
import { execFileSync } from "node:child_process";

import { SpreadsheetFile, Workbook } from "@oai/artifact-tool";

const defaultDbUrl = "postgres://cims:cims@localhost:5432/cims?sslmode=disable";
const outputDir = path.resolve(process.argv[2] || "outputs/db-table-export");
const previewDir = path.join(outputDir, "previews");
const dbUrl = process.env.DB_URL || defaultDbUrl;

function runPsql(sql, { csv = false } = {}) {
  const args = [dbUrl];
  if (csv) {
    args.push("--csv");
  } else {
    args.push("-At", "-F", "\t");
  }
  args.push("-c", sql);
  return execFileSync("psql", args, {
    encoding: "utf8",
    maxBuffer: 1024 * 1024 * 512,
  });
}

function listTables() {
  const sql = `
    select table_name
    from information_schema.tables
    where table_schema = 'public'
      and table_type = 'BASE TABLE'
    order by table_name
  `;
  return runPsql(sql)
    .split(/\r?\n/)
    .map((value) => value.trim())
    .filter(Boolean);
}

function tableCount(tableName) {
  const sql = `select count(*) from "${tableName}"`;
  return Number(runPsql(sql).trim() || "0");
}

function tableCsv(tableName) {
  const sql = `select * from "${tableName}"`;
  return runPsql(sql, { csv: true });
}

function headerNames(csvText) {
  const firstLine = csvText.split(/\r?\n/, 1)[0] || "";
  return firstLine
    .split(",")
    .map((value) => value.replace(/^"|"$/g, "").trim());
}

function columnName(index) {
  let value = "";
  let current = index + 1;
  while (current > 0) {
    const remainder = (current - 1) % 26;
    value = String.fromCharCode(65 + remainder) + value;
    current = Math.floor((current - 1) / 26);
  }
  return value;
}

function parseCsv(csvText) {
  const rows = [];
  let row = [];
  let field = "";
  let i = 0;
  let inQuotes = false;

  while (i < csvText.length) {
    const char = csvText[i];
    const next = csvText[i + 1];

    if (inQuotes) {
      if (char === '"' && next === '"') {
        field += '"';
        i += 2;
        continue;
      }
      if (char === '"') {
        inQuotes = false;
        i += 1;
        continue;
      }
      field += char;
      i += 1;
      continue;
    }

    if (char === '"') {
      inQuotes = true;
      i += 1;
      continue;
    }
    if (char === ",") {
      row.push(field);
      field = "";
      i += 1;
      continue;
    }
    if (char === "\n") {
      row.push(field);
      rows.push(row);
      row = [];
      field = "";
      i += 1;
      continue;
    }
    if (char === "\r") {
      i += 1;
      continue;
    }
    field += char;
    i += 1;
  }

  if (field.length > 0 || row.length > 0) {
    row.push(field);
    rows.push(row);
  }

  return rows;
}

function columnWidthForHeader(header) {
  const name = header.toLowerCase();
  if (name === "payload") return 26;
  if (name.includes("description") || name.includes("address")) return 24;
  if (name.includes("remarks") || name.includes("reference")) return 22;
  if (name.includes("name")) return 20;
  if (name.includes("date")) return 14;
  if (name.includes("amount") || name.includes("cost") || name.includes("price") || name.includes("balance")) return 14;
  if (name === "id" || name.endsWith("_id") || name.includes("code")) return 12;
  return 16;
}

async function styleDataSheet(sheet, headers) {
  const used = sheet.getUsedRange();
  sheet.freezePanes.freezeRows(1);
  used.getRow(0).format = {
    fill: "#1F4E78",
    font: { bold: true, color: "#FFFFFF" },
    wrapText: true,
    borders: { preset: "all", style: "thin", color: "#D9E2F3" },
  };

  for (let i = 0; i < headers.length; i += 1) {
    sheet.getRangeByIndexes(0, i, 1, 1).format.columnWidth = columnWidthForHeader(headers[i]);
  }
}

async function buildWorkbook() {
  await fs.mkdir(outputDir, { recursive: true });
  await fs.mkdir(previewDir, { recursive: true });

  const workbook = Workbook.create();
  const summary = workbook.worksheets.add("Summary");
  summary.showGridLines = false;

  const exportedAt = new Date();
  const tables = listTables();
  const summaryRows = [];

  for (const tableName of tables) {
    const count = tableCount(tableName);
    const csvText = tableCsv(tableName);
    const headers = headerNames(csvText);
    summaryRows.push([tableName, count, headers.length]);

    const matrix = parseCsv(csvText);
    const sheet = workbook.worksheets.add(tableName);
    if (matrix.length > 0 && headers.length > 0) {
      const rangeRef = `A1:${columnName(headers.length - 1)}${matrix.length}`;
      sheet.getRange(rangeRef).values = matrix;
    }
    await styleDataSheet(sheet, headers);

    const preview = await workbook.render({
      sheetName: tableName,
      range: `A1:${columnName(Math.min(Math.max(headers.length, 1), 8) - 1)}18`,
      scale: 1,
      format: "png",
    });
    await fs.writeFile(
      path.join(previewDir, `${tableName}.png`),
      new Uint8Array(await preview.arrayBuffer()),
    );
  }

  const totalRows = summaryRows.reduce((sum, [, count]) => sum + count, 0);

  summary.getRange("A1:G1").merge();
  summary.getRange("A1:G1").values = [["CIMS Database Table Export"]];
  summary.getRange("A1:G1").format = {
    fill: "#0F172A",
    font: { bold: true, color: "#FFFFFF", size: 16 },
  };

  summary.getRange("A3:B6").values = [
    ["Database", dbUrl],
    ["Exported At", exportedAt],
    ["Tables", tables.length],
    ["Total Rows", totalRows],
  ];
  summary.getRange("A3:A6").format = {
    font: { bold: true, color: "#0F172A" },
    fill: "#E2E8F0",
  };
  summary.getRange("B4").setNumberFormat("yyyy-mm-dd hh:mm");
  summary.getRange("B5:B6").format.numberFormat = "0";

  const tableStartRow = 8;
  summary.getRange(`A${tableStartRow}:C${tableStartRow}`).values = [["Table", "Rows", "Columns"]];
  summary.getRange(`A${tableStartRow}:C${tableStartRow}`).format = {
    fill: "#1D4ED8",
    font: { bold: true, color: "#FFFFFF" },
    borders: { preset: "all", style: "thin", color: "#BFDBFE" },
  };
  summary
    .getRangeByIndexes(tableStartRow, 0, summaryRows.length, 3)
    .values = summaryRows;
  summary.getRange(`B${tableStartRow + 1}:C${tableStartRow + summaryRows.length}`).format.numberFormat = "0";
  summary.getRange(`A${tableStartRow}:C${tableStartRow + summaryRows.length}`).format.borders = {
    preset: "all",
    style: "thin",
    color: "#CBD5E1",
  };
  summary.freezePanes.freezeRows(tableStartRow);

  summary.getRange("E3:G5").merge();
  summary.getRange("E3:G5").values = [[
    "Each worksheet contains a raw table export from the current Postgres database. JSON payload columns were kept as-is.",
  ]];
  summary.getRange("E3:G5").format = {
    fill: "#EFF6FF",
    font: { color: "#1E3A8A" },
    wrapText: true,
    borders: { preset: "all", style: "thin", color: "#93C5FD" },
  };

  const chartEndRow = tableStartRow + summaryRows.length;
  const chart = summary.charts.add("bar", summary.getRange(`A${tableStartRow}:B${chartEndRow}`));
  chart.title = "Rows Per Table";
  chart.hasLegend = false;
  chart.xAxis = { axisType: "textAxis" };
  chart.yAxis = { numberFormatCode: "0" };
  chart.setPosition("E8", "L24");

  summary.getRange("A1:L24").format.wrapText = true;
  summary.getRange("A:A").format.columnWidth = 24;
  summary.getRange("B:B").format.columnWidth = 18;
  summary.getRange("C:C").format.columnWidth = 12;
  summary.getRange("E:L").format.columnWidth = 14;

  const summaryPreview = await workbook.render({
    sheetName: "Summary",
    autoCrop: "all",
    scale: 1,
    format: "png",
  });
  await fs.writeFile(
    path.join(previewDir, "Summary.png"),
    new Uint8Array(await summaryPreview.arrayBuffer()),
  );

  const inspect = await workbook.inspect({
    kind: "table",
    range: `Summary!A1:C${chartEndRow}`,
    include: "values,formulas",
    tableMaxRows: Math.min(chartEndRow, 25),
    tableMaxCols: 6,
  });
  console.log(inspect.ndjson);

  const errors = await workbook.inspect({
    kind: "match",
    searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A",
    options: { useRegex: true, maxResults: 100 },
    summary: "final formula error scan",
  });
  console.log(errors.ndjson);

  const output = await SpreadsheetFile.exportXlsx(workbook);
  const outputPath = path.join(outputDir, "cims-database-tables.xlsx");
  await output.save(outputPath);
  console.log(`saved=${outputPath}`);
}

await buildWorkbook();
