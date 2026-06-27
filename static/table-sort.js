(function () {
  if (window.CIMSTableSorterLoaded) return;
  window.CIMSTableSorterLoaded = true;

  var sortableTableSelector = [
    "table[data-fill-empty-rows]",
    ".form-backdrop-table > table",
    ".report-table"
  ].join(",");

  function toArray(value) {
    return Array.prototype.slice.call(value || []);
  }

  function textForCell(cell) {
    if (!cell) return "";
    var explicit = cell.getAttribute("data-sort-value");
    if (explicit !== null) return explicit;
    return (cell.innerText || cell.textContent || "").replace(/\s+/g, " ").trim();
  }

  function parseSortableNumber(text) {
    var raw = String(text || "").trim();
    if (!raw) return null;
    var cleaned = raw.replace(/,/g, "").replace(/\s/g, "").replace(/%$/, "");
    cleaned = cleaned.replace(/^\$/, "");
    if (/^\(.*\)$/.test(cleaned)) {
      cleaned = "-" + cleaned.slice(1, -1);
    }
    if (!/^-?\d+(\.\d+)?$/.test(cleaned)) return null;
    var value = Number(cleaned);
    return Number.isFinite(value) ? value : null;
  }

  function parseSortableDate(text) {
    var raw = String(text || "").trim();
    var match = raw.match(/^(\d{4})-(\d{1,2})-(\d{1,2})(?:\s|$)/);
    if (match) {
      return Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
    }

    match = raw.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})(?:,\s*(\d{1,2}):(\d{2})\s*(AM|PM))?/i);
    if (!match) return null;

    var hour = Number(match[4] || 0);
    var minute = Number(match[5] || 0);
    var meridiem = String(match[6] || "").toUpperCase();
    if (meridiem === "PM" && hour < 12) hour += 12;
    if (meridiem === "AM" && hour === 12) hour = 0;
    return Date.UTC(Number(match[3]), Number(match[1]) - 1, Number(match[2]), hour, minute);
  }

  function sortableValue(row, columnIndex) {
    var cell = cellAtColumn(row, columnIndex);
    var text = textForCell(cell);
    if (!text) return { type: "empty", value: "", text: "" };

    var date = parseSortableDate(text);
    if (date !== null) return { type: "date", value: date, text: text };

    var number = parseSortableNumber(text);
    if (number !== null) return { type: "number", value: number, text: text };

    return { type: "text", value: text.toLocaleLowerCase(), text: text };
  }

  function compareValues(left, right) {
    if (left.type === "empty" || right.type === "empty") {
      if (left.type === right.type) return 0;
      return left.type === "empty" ? 1 : -1;
    }
    if (left.type === right.type && (left.type === "number" || left.type === "date")) {
      return left.value === right.value ? 0 : left.value < right.value ? -1 : 1;
    }
    return left.text.localeCompare(right.text, undefined, { numeric: true, sensitivity: "base" });
  }

  function cellAtColumn(row, columnIndex) {
    var logicalIndex = 0;
    for (var index = 0; index < row.cells.length; index++) {
      var cell = row.cells[index];
      var span = Math.max(1, cell.colSpan || 1);
      if (columnIndex >= logicalIndex && columnIndex < logicalIndex + span) {
        return cell;
      }
      logicalIndex += span;
    }
    return null;
  }

  function rowCanSort(row, columnIndex) {
    if (!row || row.parentElement.tagName !== "TBODY") return false;
    if (row.classList.contains("add-new-row") || row.classList.contains("empty-fill-row") || row.classList.contains("lazy-load-row")) return false;
    if (row.querySelector("th")) return false;
    if (row.querySelector("input, textarea, select, [contenteditable='true']")) return false;
    if (row.querySelector(".empty")) return false;

    var cell = cellAtColumn(row, columnIndex);
    if (!cell || cell.tagName !== "TD") return false;

    return !toArray(row.cells).some(function (candidate) {
      return (candidate.colSpan || 1) > 1;
    });
  }

  function sortBody(tbody, columnIndex, direction) {
    toArray(tbody.querySelectorAll(".empty-fill-row")).forEach(function (row) { row.remove(); });

    var fragment = document.createDocumentFragment();
    var group = [];

    function appendSortedGroup() {
      if (!group.length) return;
      group.sort(function (left, right) {
        var result = compareValues(left.value, right.value);
        if (result === 0) result = left.index - right.index;
        return direction === "desc" ? -result : result;
      });
      group.forEach(function (item) {
        fragment.appendChild(item.row);
      });
      group = [];
    }

    toArray(tbody.rows).forEach(function (row, index) {
      if (rowCanSort(row, columnIndex)) {
        group.push({ row: row, value: sortableValue(row, columnIndex), index: index });
        return;
      }
      appendSortedGroup();
      fragment.appendChild(row);
    });
    appendSortedGroup();
    tbody.appendChild(fragment);
  }

  function headerColumnIndex(targetHeader) {
    var thead = targetHeader.closest("thead");
    if (!thead) return -1;

    var occupied = [];
    var found = -1;
    toArray(thead.rows).some(function (row, rowIndex) {
      occupied[rowIndex] = occupied[rowIndex] || [];
      var columnIndex = 0;
      return toArray(row.cells).some(function (cell) {
        while (occupied[rowIndex][columnIndex]) columnIndex += 1;

        var colSpan = Math.max(1, cell.colSpan || 1);
        var rowSpan = Math.max(1, cell.rowSpan || 1);
        if (cell === targetHeader) {
          found = columnIndex;
          return true;
        }

        for (var rowOffset = 0; rowOffset < rowSpan; rowOffset++) {
          var occupiedRow = rowIndex + rowOffset;
          occupied[occupiedRow] = occupied[occupiedRow] || [];
          for (var colOffset = 0; colOffset < colSpan; colOffset++) {
            occupied[occupiedRow][columnIndex + colOffset] = true;
          }
        }
        columnIndex += colSpan;
        return false;
      });
    });

    return found;
  }

  function updateHeaderState(table, columnIndex, direction) {
    toArray(table.tHead.querySelectorAll("th[data-sortable-column]")).forEach(function (header) {
      var active = Number(header.getAttribute("data-sortable-column")) === columnIndex;
      var ariaSort = "none";
      if (active) {
        ariaSort = direction === "desc" ? "descending" : "ascending";
      }
      header.setAttribute("aria-sort", ariaSort);
      if (active) {
        header.setAttribute("data-sort-direction", direction);
      } else {
        header.removeAttribute("data-sort-direction");
      }
    });
  }

  function applyTableSort(table, columnIndex, direction) {
    if (!table.tBodies || !table.tBodies.length) return;
    table.setAttribute("data-sort-column", String(columnIndex));
    table.setAttribute("data-sort-direction", direction);
    toArray(table.tBodies).forEach(function (tbody) {
      sortBody(tbody, columnIndex, direction);
    });
    updateHeaderState(table, columnIndex, direction);
    if (typeof window.CIMSFillListTables === "function") {
      window.CIMSFillListTables();
    }
  }

  function nextDirection(table, columnIndex) {
    var currentColumn = Number(table.getAttribute("data-sort-column"));
    var currentDirection = table.getAttribute("data-sort-direction");
    return currentColumn === columnIndex && currentDirection === "asc" ? "desc" : "asc";
  }

  function tableHasSortableRow(table, columnIndex) {
    return toArray(table.tBodies).some(function (tbody) {
      return toArray(tbody.rows).some(function (row) {
        return rowCanSort(row, columnIndex);
      });
    });
  }

  function initTable(table) {
    if (!table || !table.tHead || !table.tBodies || !table.tBodies.length) return;

    toArray(table.tHead.querySelectorAll("th")).forEach(function (header) {
      if (header.getAttribute("data-sortable-column") !== null) return;
      if ((header.colSpan || 1) > 1 || !textForCell(header)) return;

      var columnIndex = headerColumnIndex(header);
      if (columnIndex < 0 || !tableHasSortableRow(table, columnIndex)) return;

      header.setAttribute("data-sortable-column", String(columnIndex));
      header.setAttribute("aria-sort", "none");
      header.tabIndex = 0;
      header.title = header.title ? header.title + " Sort" : "Sort";
      header.addEventListener("click", function () {
        applyTableSort(table, columnIndex, nextDirection(table, columnIndex));
      });
      header.addEventListener("keydown", function (event) {
        if (event.key !== "Enter" && event.key !== " ") return;
        event.preventDefault();
        applyTableSort(table, columnIndex, nextDirection(table, columnIndex));
      });
    });
  }

  function initSortableTables(root) {
    toArray((root || document).querySelectorAll(sortableTableSelector)).forEach(initTable);
  }

  function reapplyActiveSorts(root) {
    toArray((root || document).querySelectorAll(sortableTableSelector)).forEach(function (table) {
      var column = table.getAttribute("data-sort-column");
      var direction = table.getAttribute("data-sort-direction");
      if (column === null || (direction !== "asc" && direction !== "desc")) return;
      applyTableSort(table, Number(column), direction);
    });
  }

  function refreshSortableTables() {
    initSortableTables(document);
    reapplyActiveSorts(document);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", refreshSortableTables);
  } else {
    refreshSortableTables();
  }
  document.addEventListener("htmx:afterSettle", refreshSortableTables);
})();
