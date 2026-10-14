// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var completed = task.classList.toggle("task-completed");
  var button = form.querySelector("button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }

  // La tarea pasa al principio del bloque de destino.
  var target = document.querySelector('[data-task-list="' + (completed ? "completed" : "pending") + '"]');
  if (target) {
    target.insertBefore(task, target.firstElementChild);
  }

  fetch(form.action);
});

// Reordena las tareas de un bloque arrastrándolas con el ratón.
(function () {
  var DRAG_THRESHOLD = 4;
  var DROP_FALLBACK_MS = 250;

  var drag = null;
  var frame = null;

  function prefersReducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function tasksOf(list) {
    return Array.prototype.filter.call(list.children, function (el) {
      return el.classList.contains("task");
    });
  }

  function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
  }

  function start() {
    var item = drag.item;
    var list = drag.list;

    try {
      item.setPointerCapture(drag.pointerId);
    } catch (e) {
      // El puntero puede haber desaparecido; el arrastre sigue con document.
    }

    var selection = window.getSelection && window.getSelection();
    if (selection && !selection.isCollapsed) {
      selection.removeAllRanges();
    }

    var all = tasksOf(list);
    var itemRect = item.getBoundingClientRect();
    var listRect = list.getBoundingClientRect();

    drag.startIndex = all.indexOf(item);
    drag.targetIndex = drag.startIndex;
    drag.siblings = all.filter(function (el) {
      return el !== item;
    });
    drag.metrics = drag.siblings.map(function (el) {
      var rect = el.getBoundingClientRect();
      return { center: rect.top + rect.height / 2, height: rect.height };
    });
    drag.itemHeight = item.offsetHeight;
    drag.itemCenter = itemRect.top + itemRect.height / 2;
    drag.minDy = listRect.top - itemRect.top;
    drag.maxDy = listRect.bottom - itemRect.bottom;
    drag.started = true;

    item.classList.add("is-dragging");
    list.classList.add("is-sorting");
    document.body.classList.add("is-dragging-task");
  }

  function computeTargetIndex() {
    var center = drag.itemCenter + drag.dy;
    var index = 0;
    drag.metrics.forEach(function (m) {
      if (m.center < center) {
        index++;
      }
    });
    return index;
  }

  function render() {
    frame = null;
    if (!drag || !drag.started || drag.dropping) {
      return;
    }

    drag.item.style.transform = "translateY(" + drag.dy + "px) scale(1.02)";
    drag.targetIndex = computeTargetIndex();

    var startIndex = drag.startIndex;
    var targetIndex = drag.targetIndex;
    var shift = drag.itemHeight;
    drag.siblings.forEach(function (el, i) {
      if (i < startIndex && i >= targetIndex) {
        el.style.transform = "translateY(" + shift + "px)";
      } else if (i >= startIndex && i < targetIndex) {
        el.style.transform = "translateY(" + -shift + "px)";
      } else {
        el.style.transform = "";
      }
    });
  }

  function scheduleRender() {
    if (frame === null) {
      frame = window.requestAnimationFrame(render);
    }
  }

  function flushRender() {
    if (frame !== null) {
      window.cancelAnimationFrame(frame);
    }
    render();
  }

  function afterTransition(el, done) {
    var called = false;
    var timer = null;

    function run() {
      if (called) {
        return;
      }
      called = true;
      el.removeEventListener("transitionend", onEnd);
      clearTimeout(timer);
      done();
    }

    function onEnd(event) {
      if (event.target === el && event.propertyName === "transform") {
        run();
      }
    }

    if (prefersReducedMotion()) {
      run();
      return;
    }

    el.addEventListener("transitionend", onEnd);
    timer = setTimeout(run, DROP_FALLBACK_MS);
  }

  function dropOffset() {
    var offset = 0;
    var i;
    if (drag.targetIndex > drag.startIndex) {
      for (i = drag.startIndex; i < drag.targetIndex; i++) {
        offset += drag.metrics[i].height;
      }
    } else if (drag.targetIndex < drag.startIndex) {
      for (i = drag.targetIndex; i < drag.startIndex; i++) {
        offset -= drag.metrics[i].height;
      }
    }
    return offset;
  }

  function finish(move) {
    var state = drag;
    var item = state.item;
    var list = state.list;

    list.classList.remove("is-sorting");
    item.classList.remove("is-dragging", "is-dropping");
    document.body.classList.remove("is-dragging-task");
    item.style.transform = "";
    state.siblings.forEach(function (el) {
      el.style.transform = "";
    });

    var moved = move && state.targetIndex !== state.startIndex;
    if (moved) {
      list.insertBefore(item, state.siblings[state.targetIndex] || null);
    }
    void list.offsetHeight;

    drag = null;

    if (moved) {
      persist(list);
    }
  }

  function drop() {
    flushRender();
    drag.dropping = true;
    drag.item.classList.add("is-dropping");
    drag.item.style.transform = "translateY(" + dropOffset() + "px)";
    afterTransition(drag.item, function () {
      finish(true);
    });
  }

  function cancel() {
    if (!drag || drag.dropping) {
      return;
    }
    if (!drag.started) {
      drag = null;
      return;
    }

    if (frame !== null) {
      window.cancelAnimationFrame(frame);
      frame = null;
    }
    drag.dropping = true;
    drag.targetIndex = drag.startIndex;
    drag.item.classList.add("is-dropping");
    drag.item.style.transform = "translateY(0px)";
    drag.siblings.forEach(function (el) {
      el.style.transform = "translateY(0px)";
    });
    afterTransition(drag.item, function () {
      finish(false);
    });
  }

  function suppressNextClick() {
    function onClick(event) {
      event.preventDefault();
      event.stopPropagation();
    }
    document.addEventListener("click", onClick, true);
    setTimeout(function () {
      document.removeEventListener("click", onClick, true);
    }, 0);
  }

  function persist(list) {
    var groups = document.querySelector("[data-task-groups]");
    if (!groups) {
      return;
    }

    var token = groups.dataset.csrfToken;
    var body = new URLSearchParams();
    body.append("authenticity_token", token);
    body.append("block", list.dataset.taskList);
    tasksOf(list).forEach(function (el) {
      body.append("ids", el.dataset.taskId);
    });

    function reload() {
      window.location.reload();
    }

    fetch(groups.dataset.reorderUrl, {
      method: "POST",
      body: body,
      credentials: "same-origin",
      headers: { "X-CSRF-Token": token }
    }).then(function (response) {
      if (!response.ok) {
        reload();
      }
    }, reload);
  }

  document.addEventListener("pointerdown", function (event) {
    // Un arrastre a la vez. Uno que no llegó a empezar (p. ej. se soltó fuera
    // de la ventana) se descarta.
    if ((drag && drag.started) || event.pointerType !== "mouse" || event.button !== 0) {
      return;
    }

    var item = event.target.closest(".task-list .task");
    if (!item || event.target.closest("button, a, input, select, textarea")) {
      return;
    }

    // Evita los eventos de ratón de compatibilidad, pero no basta para impedir
    // la selección de texto: eso lo hacen el CSS (user-select) y los listeners
    // de selectstart y dragstart.
    event.preventDefault();
    drag = {
      item: item,
      list: item.parentElement,
      pointerId: event.pointerId,
      startY: event.clientY,
      dy: 0,
      started: false,
      dropping: false
    };
  });

  document.addEventListener("pointermove", function (event) {
    if (!drag || drag.dropping || event.pointerId !== drag.pointerId) {
      return;
    }

    var dy = event.clientY - drag.startY;
    if (!drag.started) {
      if (Math.abs(dy) < DRAG_THRESHOLD) {
        return;
      }
      start();
    }

    drag.dy = clamp(dy, drag.minDy, drag.maxDy);
    scheduleRender();
  });

  document.addEventListener("pointerup", function (event) {
    if (!drag || drag.dropping || event.pointerId !== drag.pointerId) {
      return;
    }

    if (!drag.started) {
      drag = null;
      return;
    }

    suppressNextClick();
    drop();
  });

  document.addEventListener("pointercancel", function (event) {
    if (drag && event.pointerId === drag.pointerId) {
      cancel();
    }
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape") {
      cancel();
    }
  });

  window.addEventListener("blur", cancel);

  document.addEventListener("selectstart", function (event) {
    if (drag) {
      event.preventDefault();
    }
  });

  document.addEventListener("dragstart", function (event) {
    if (drag) {
      event.preventDefault();
    }
  });
})();
