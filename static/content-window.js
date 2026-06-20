(function () {
  if (window.CIMSContentWindowManagerLoaded) return;
  window.CIMSContentWindowManagerLoaded = true;

  var primaryRoot = document.querySelector("[data-content-window]");

  var zIndex = 60;
  var windowCount = 0;
  var NAV_HEIGHT = 84;
  var SIDEBAR_WIDTH = 250;
  var taskbar = null;

  function sidebarLeft() {
    return document.body.classList.contains("sidebar-collapsed") ? 0 : SIDEBAR_WIDTH;
  }

  function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
  }

  function taskbarHeight() {
    if (!taskbar || taskbar.hidden) return 0;
    return taskbar.getBoundingClientRect().height;
  }

  function workspaceBounds() {
    return {
      left: sidebarLeft(),
      top: NAV_HEIGHT,
      right: window.innerWidth,
      bottom: window.innerHeight - taskbarHeight()
    };
  }

  function constrainRect(rect) {
    var bounds = workspaceBounds();
    var maxWidth = Math.max(0, bounds.right - bounds.left);
    var maxHeight = Math.max(0, bounds.bottom - bounds.top);
    var minWidth = Math.min(520, maxWidth);
    var minHeight = Math.min(360, maxHeight);
    var width = clamp(rect.width, minWidth, maxWidth);
    var height = clamp(rect.height, minHeight, maxHeight);
    var left = clamp(rect.left, bounds.left, Math.max(bounds.left, bounds.right - width));
    var top = clamp(rect.top, bounds.top, Math.max(bounds.top, bounds.bottom - height));

    return {
      left: left,
      top: top,
      width: width,
      height: height
    };
  }

  function keepWindowInWorkspace(root) {
    if (!root || root.classList.contains("is-page-minimized")) return;
    if (!root.classList.contains("is-page-floating")) return;
    pinWindowToRect(root, root.getBoundingClientRect());
  }

  function keepFloatingWindowsInWorkspace() {
    document.querySelectorAll(".content-window.is-page-floating").forEach(keepWindowInWorkspace);
  }

  function makeButton(className, attr, title) {
    var button = document.createElement("button");
    button.type = "button";
    button.className = "desktop-window-control " + className;
    button.setAttribute(attr, "");
    button.title = title;
    button.setAttribute("aria-label", title);
    return button;
  }

  function ensureTaskbar() {
    if (taskbar) return taskbar;

    taskbar = document.createElement("div");
    taskbar.className = "content-window-taskbar";
    taskbar.setAttribute("data-content-window-taskbar", "");
    taskbar.hidden = true;
    document.body.appendChild(taskbar);
    return taskbar;
  }

  function refreshTaskbar() {
    if (!taskbar) return;

    var hasItems = taskbar.children.length > 0;
    taskbar.hidden = !hasItems;
    document.body.classList.toggle("has-window-taskbar", hasItems);
    keepFloatingWindowsInWorkspace();
  }

  function windowTitle(root) {
    var title = root.querySelector(".content-window-title");
    var text = title ? title.textContent.replace(/\s+/g, " ").trim() : "";
    return text || "Window";
  }

  function addResizeHandle(root, edge) {
    var handle = root.querySelector("[data-content-window-resize='" + edge + "']");
    if (!handle) {
      handle = document.createElement("div");
      handle.className = "content-window-resize-handle content-window-resize-" + edge;
      handle.setAttribute("data-content-window-resize", edge);
      handle.setAttribute("aria-hidden", "true");
      root.appendChild(handle);
    }
    return handle;
  }

  function pinWindowToRect(root, rect) {
    var constrained = constrainRect(rect);
    root.style.position = "fixed";
    root.style.left = constrained.left + "px";
    root.style.top = constrained.top + "px";
    root.style.width = constrained.width + "px";
    root.style.height = constrained.height + "px";
    root.style.margin = "0";
    root.classList.add("is-page-floating");
  }

  function focusWindow(root) {
    zIndex += 1;
    root.style.zIndex = zIndex;
  }

  function isDarkTheme() {
    return document.documentElement.classList.contains("theme-dark");
  }

  function desktopBackground() {
    return isDarkTheme() ? "#202F46" : "#9DB9DB";
  }

  function childFrameChromeCSS() {
    var background = desktopBackground();
    return [
      ".navrail,.sidebar,.content-window-titlebar,.content-window-restore,.content-window-resize-handle{display:none!important}",
      "html,body{width:100%!important;height:100%!important;min-height:100%!important;background:" + background + "!important;overflow:hidden!important}",
      ".page{width:100%!important;height:100vh!important;margin:0!important;margin-top:0!important;min-height:100vh!important;padding:0!important;background:" + background + "!important;overflow:hidden!important}",
      ".content-window{position:relative!important;left:0!important;top:0!important;right:auto!important;bottom:auto!important;width:100%!important;height:100vh!important;min-width:0!important;min-height:0!important;max-width:none!important;max-height:none!important;margin:0!important;padding:0!important;overflow:hidden!important;background:" + background + "!important}",
      ".content-window.is-page-maximized{position:relative!important;left:0!important;top:0!important;right:auto!important;bottom:auto!important;width:100%!important;height:100vh!important}",
      ".content-window-body{width:100%!important;height:100%!important;overflow:auto!important}",
      ".report-viewer-page,.report-options-page{width:100%!important;height:100vh!important;min-height:100vh!important;overflow:hidden!important}",
      ".report-viewer-shell{width:100%!important;height:100vh!important;min-height:100vh!important}",
      ".report-preview-tree,.report-preview{height:100vh!important;min-height:100vh!important}",
      ".report-paper-scroll{height:100%!important;min-height:0!important}",
      ".table-wrap{margin:0!important}"
    ].join("\n");
  }

  function syncChildFrameTheme(iframe) {
    try {
      var doc = iframe.contentDocument;
      if (!doc || !doc.head || !doc.documentElement) return;

      doc.documentElement.classList.toggle("theme-dark", isDarkTheme());

      var style = doc.querySelector("style[data-child-window-chrome]");
      if (!style) {
        style = doc.createElement("style");
        style.setAttribute("data-child-window-chrome", "");
        doc.head.appendChild(style);
      }
      style.textContent = childFrameChromeCSS();
    } catch (error) {}
  }

  function initContentWindow(root, options) {
    var managed = options && options.managed;
    var startMaximized = options && options.startMaximized;
    var minimize = root.querySelector("[data-content-window-minimize]");
    var maximize = root.querySelector("[data-content-window-maximize]");
    var close = root.querySelector("[data-content-window-close]");
    var restore = root.querySelector("[data-content-window-restore]");
    var titlebar = root.querySelector(".content-window-titlebar");
    var resizeHandles = [
      addResizeHandle(root, "right"),
      addResizeHandle(root, "bottom"),
      addResizeHandle(root, "corner")
    ];
    var dragState = null;
    var resizeState = null;
    var restoreRect = null;
    var taskbarItem = null;

    function removeTaskbarItem() {
      if (!taskbarItem) return;
      taskbarItem.remove();
      taskbarItem = null;
      refreshTaskbar();
    }

    function ensureTaskbarItem() {
      if (taskbarItem) return;

      var bar = ensureTaskbar();
      var item = document.createElement("div");
      item.className = "content-window-taskbar-item";

      var restoreButton = document.createElement("button");
      restoreButton.type = "button";
      restoreButton.className = "content-window-taskbar-title";
      restoreButton.title = "Restore " + windowTitle(root);
      restoreButton.setAttribute("aria-label", "Restore " + windowTitle(root));

      var icon = document.createElement("span");
      icon.className = "taskbar-window-icon";
      icon.setAttribute("aria-hidden", "true");

      var label = document.createElement("span");
      label.className = "content-window-taskbar-label";
      label.textContent = windowTitle(root);

      restoreButton.appendChild(icon);
      restoreButton.appendChild(label);
      restoreButton.addEventListener("click", function () {
        setMinimized(false);
      });

      var controls = document.createElement("div");
      controls.className = "content-window-taskbar-controls";
      var restoreControl = makeButton("desktop-window-restore-control taskbar-window-control", "data-taskbar-window-restore", "Restore");
      var maximizeControl = makeButton("desktop-window-maximize taskbar-window-control", "data-taskbar-window-maximize", "Maximize");
      var closeControl = makeButton("desktop-window-close taskbar-window-control", "data-taskbar-window-close", "Close");

      restoreControl.addEventListener("click", function (event) {
        event.stopPropagation();
        setMinimized(false);
      });
      maximizeControl.addEventListener("click", function (event) {
        event.stopPropagation();
        setMaximized(true);
      });
      closeControl.addEventListener("click", function (event) {
        event.stopPropagation();
        closeWindow();
      });

      controls.appendChild(restoreControl);
      controls.appendChild(maximizeControl);
      controls.appendChild(closeControl);
      item.appendChild(restoreButton);
      item.appendChild(controls);
      bar.appendChild(item);
      taskbarItem = item;
      refreshTaskbar();
    }

    function setMinimized(minimized) {
      root.classList.toggle("is-page-minimized", minimized);
      if (minimized) {
        ensureTaskbarItem();
      } else {
        removeTaskbarItem();
      }
      if (restore) restore.hidden = true;
      if (!minimized) focusWindow(root);
    }

    function setMaximized(maximized) {
      if (maximized) {
        restoreRect = root.getBoundingClientRect();
        root.classList.remove("is-page-floating");
        root.style.position = "";
        root.style.left = "";
        root.style.top = "";
        root.style.width = "";
        root.style.height = "";
        root.style.margin = "";
        setMinimized(false);
      } else if (restoreRect) {
        pinWindowToRect(root, restoreRect);
        restoreRect = null;
      }

      root.classList.toggle("is-page-maximized", maximized);
      if (maximize) {
        maximize.setAttribute("aria-label", maximized ? "Restore" : "Maximize");
        maximize.title = maximized ? "Restore" : "Maximize";
      }
      focusWindow(root);
    }

    function closeWindow() {
      removeTaskbarItem();
      if (managed) {
        root.remove();
      } else if (document.querySelector("[data-managed-window]")) {
        root.remove();
      } else {
        window.location.href = "/dashboard";
      }
    }

    function startDrag(event) {
      if (event.button !== undefined && event.button !== 0) return;
      if (event.target.closest(".content-window-controls")) return;
      if (root.classList.contains("is-page-maximized")) return;

      var rect = root.getBoundingClientRect();
      pinWindowToRect(root, rect);
      rect = root.getBoundingClientRect();
      dragState = {
        offsetX: event.clientX - rect.left,
        offsetY: event.clientY - rect.top,
        width: rect.width,
        height: rect.height
      };
      focusWindow(root);
      event.preventDefault();
    }

    function moveDrag(event) {
      if (!dragState) return;

      var bounds = workspaceBounds();
      var maxLeft = Math.max(bounds.left, bounds.right - dragState.width);
      var maxTop = Math.max(bounds.top, bounds.bottom - dragState.height);
      root.style.left = clamp(event.clientX - dragState.offsetX, bounds.left, maxLeft) + "px";
      root.style.top = clamp(event.clientY - dragState.offsetY, bounds.top, maxTop) + "px";
    }

    function stopDrag() {
      dragState = null;
    }

    function startResize(event) {
      if (event.button !== undefined && event.button !== 0) return;
      if (root.classList.contains("is-page-maximized")) return;

      var rect = root.getBoundingClientRect();
      var edge = event.currentTarget.getAttribute("data-content-window-resize") || "corner";
      pinWindowToRect(root, rect);
      rect = root.getBoundingClientRect();
      resizeState = {
        edge: edge,
        startX: event.clientX,
        startY: event.clientY,
        width: rect.width,
        height: rect.height,
        left: rect.left,
        top: rect.top
      };
      focusWindow(root);
      event.preventDefault();
    }

    function moveResize(event) {
      if (!resizeState) return;

      var bounds = workspaceBounds();
      var maxWidth = Math.max(0, bounds.right - resizeState.left);
      var maxHeight = Math.max(0, bounds.bottom - resizeState.top);
      var minWidth = Math.min(520, maxWidth);
      var minHeight = Math.min(360, maxHeight);
      var width = resizeState.width + event.clientX - resizeState.startX;
      var height = resizeState.height + event.clientY - resizeState.startY;

      if (resizeState.edge === "right" || resizeState.edge === "corner") {
        root.style.width = clamp(width, minWidth, maxWidth) + "px";
      }
      if (resizeState.edge === "bottom" || resizeState.edge === "corner") {
        root.style.height = clamp(height, minHeight, maxHeight) + "px";
      }
    }

    function stopResize() {
      resizeState = null;
    }

    root.addEventListener("mousedown", function () {
      focusWindow(root);
    });

    if (restore) {
      restore.hidden = true;
      restore.addEventListener("click", function () {
        setMinimized(false);
      });
    }

    if (minimize) {
      minimize.addEventListener("click", function () {
        setMinimized(true);
      });
    }

    if (maximize) {
      maximize.addEventListener("click", function () {
        setMaximized(!root.classList.contains("is-page-maximized"));
      });
    }

    if (close) {
      close.addEventListener("click", closeWindow);
    }

    if (titlebar) {
      titlebar.addEventListener("pointerdown", function (event) {
        startDrag(event);
        if (!dragState) return;
        dragState.pointerId = event.pointerId;
        titlebar.setPointerCapture(event.pointerId);
      });
      titlebar.addEventListener("pointermove", function (event) {
        if (dragState && dragState.pointerId === event.pointerId) moveDrag(event);
      });
      titlebar.addEventListener("pointerup", function (event) {
        if (!dragState || dragState.pointerId !== event.pointerId) return;
        stopDrag();
        titlebar.releasePointerCapture(event.pointerId);
      });
      titlebar.addEventListener("pointercancel", function (event) {
        if (!dragState || dragState.pointerId !== event.pointerId) return;
        stopDrag();
      });
      titlebar.addEventListener("mousedown", startDrag);
    }

    resizeHandles.forEach(function (handle) {
      handle.addEventListener("pointerdown", function (event) {
        startResize(event);
        if (!resizeState) return;
        resizeState.pointerId = event.pointerId;
        handle.setPointerCapture(event.pointerId);
      });
      handle.addEventListener("pointermove", function (event) {
        if (resizeState && resizeState.pointerId === event.pointerId) moveResize(event);
      });
      handle.addEventListener("pointerup", function (event) {
        if (!resizeState || resizeState.pointerId !== event.pointerId) return;
        stopResize();
        handle.releasePointerCapture(event.pointerId);
      });
      handle.addEventListener("pointercancel", function (event) {
        if (!resizeState || resizeState.pointerId !== event.pointerId) return;
        stopResize();
      });
      handle.addEventListener("mousedown", startResize);
    });

    document.addEventListener("mousemove", function (event) {
      moveDrag(event);
      moveResize(event);
    });
    document.addEventListener("mouseup", function () {
      stopDrag();
      stopResize();
      keepWindowInWorkspace(root);
    });

    if (startMaximized) {
      setMaximized(true);
    } else {
      focusWindow(root);
    }
  }

  function injectChildFrameChrome(iframe) {
    syncChildFrameTheme(iframe);
  }

  function windowTitleFromLink(link, url) {
    var text = link ? link.textContent.replace(/\s+/g, " ").trim() : "";
    if (text) return text;
    try {
      return new URL(url, window.location.href).pathname;
    } catch (error) {
      return "Window";
    }
  }

  function openPageWindow(url, title) {
    windowCount += 1;

    var bounds = workspaceBounds();
    var leftBase = bounds.left + 24 + ((windowCount - 1) % 7) * 28;
    var topBase = bounds.top + 20 + ((windowCount - 1) % 7) * 24;
    var width = Math.min(960, Math.max(560, bounds.right - leftBase - 28));
    var height = Math.min(640, Math.max(390, bounds.bottom - topBase - 28));
    var rect = constrainRect({
      left: leftBase,
      top: topBase,
      width: width,
      height: height
    });

    var root = document.createElement("section");
    root.className = "content-window desktop-page-window is-page-floating";
    root.setAttribute("data-content-window", "");
    root.setAttribute("data-managed-window", "");
    root.style.position = "fixed";
    root.style.left = rect.left + "px";
    root.style.top = rect.top + "px";
    root.style.width = rect.width + "px";
    root.style.height = rect.height + "px";
    root.style.margin = "0";

    var titlebar = document.createElement("div");
    titlebar.className = "content-window-titlebar";
    var titleNode = document.createElement("span");
    titleNode.className = "content-window-title";
    titleNode.textContent = title || "Window";
    var controls = document.createElement("div");
    controls.className = "content-window-controls";
    controls.appendChild(makeButton("desktop-window-minimize", "data-content-window-minimize", "Minimize"));
    controls.appendChild(makeButton("desktop-window-maximize", "data-content-window-maximize", "Maximize"));
    controls.appendChild(makeButton("desktop-window-close", "data-content-window-close", "Close"));
    titlebar.appendChild(titleNode);
    titlebar.appendChild(controls);

    var body = document.createElement("div");
    body.className = "content-window-body desktop-page-window-body";
    body.setAttribute("data-content-window-body", "");

    var iframe = document.createElement("iframe");
    iframe.className = "desktop-page-frame";
    iframe.src = url;
    iframe.title = title || "Window";
    iframe.addEventListener("load", function () {
      injectChildFrameChrome(iframe);
      try {
        var childTitle = iframe.contentDocument.title.replace(/\s+·\s+CIMS$/, "");
        if (childTitle) {
          titleNode.textContent = childTitle;
          iframe.title = childTitle;
        }
      } catch (error) {}
    });
    body.appendChild(iframe);

    root.appendChild(titlebar);
    root.appendChild(body);
    document.body.appendChild(root);
    initContentWindow(root, { managed: true, startMaximized: true });
  }

  function shouldOpenInWindow(link, event) {
    if (!link || link.target || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return false;
    if (!link.matches(".side-link[href], .dropdown-menu a[href]")) return false;

    var href = link.getAttribute("href");
    if (!href || href.charAt(0) === "#") return false;

    var url;
    try {
      url = new URL(href, window.location.href);
    } catch (error) {
      return false;
    }
    return url.origin === window.location.origin && url.pathname !== "/dashboard";
  }

  if (primaryRoot) {
    initContentWindow(primaryRoot, { managed: false, startMaximized: true });
  }

  window.addEventListener("cims:themechange", function () {
    document.querySelectorAll(".desktop-page-frame").forEach(syncChildFrameTheme);
  });

  window.addEventListener("resize", keepFloatingWindowsInWorkspace);

  new MutationObserver(keepFloatingWindowsInWorkspace).observe(document.body, {
    attributes: true,
    attributeFilter: ["class"]
  });

  document.addEventListener("click", function (event) {
    var link = event.target.closest("a[href]");
    if (!shouldOpenInWindow(link, event)) return;

    event.preventDefault();
    openPageWindow(link.href, windowTitleFromLink(link, link.href));
  });
})();
