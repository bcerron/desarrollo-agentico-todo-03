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

  fetch(form.action, { method: "POST", body: new FormData(form) });
});

// Reordenar tareas: arrastre con ratón desde el asa y Alt + flechas con teclado.
document.documentElement.classList.add("js");

(function () {
  var list = null;
  var dragged = null;
  var startOrder = "";
  var saving = false;

  function currentList() {
    return document.querySelector(".task-list[data-reorder-url]");
  }

  function taskIds(container) {
    return Array.prototype.map.call(container.querySelectorAll(".task"), function (task) {
      return task.dataset.taskId;
    });
  }

  // Guarda el orden del DOM; ante cualquier fallo recarga para recuperar el orden real.
  function save(container) {
    var tokenInput = document.querySelector("input[name=authenticity_token]");
    var body = new URLSearchParams();
    body.append("authenticity_token", tokenInput ? tokenInput.value : "");
    body.append("_method", "PUT");
    taskIds(container).forEach(function (id) {
      body.append("ids", id);
    });

    saving = true;
    fetch(container.dataset.reorderUrl, { method: "POST", body: body })
      .then(function (response) {
        if (!response.ok) {
          window.location.reload();
        }
      })
      .catch(function () {
        window.location.reload();
      })
      .then(function () {
        saving = false;
      });
  }

  function announce(task) {
    var tasks = list ? list.querySelectorAll(".task") : [];
    var position = Array.prototype.indexOf.call(tasks, task) + 1;
    var status = document.getElementById("reorder-status");
    if (status) {
      status.textContent = "Tarea movida a la posición " + position + " de " + tasks.length;
    }
  }

  document.addEventListener("mousedown", function (event) {
    var handle = event.target.closest("[data-drag-handle]");
    if (handle && !saving) {
      handle.closest(".task").setAttribute("draggable", "true");
    }
  });

  document.addEventListener("mouseup", function () {
    var task = document.querySelector(".task[draggable]");
    if (task && !dragged) {
      task.removeAttribute("draggable");
    }
  });

  document.addEventListener("dragstart", function (event) {
    var task = event.target.closest && event.target.closest(".task");
    list = currentList();
    if (!task || !list || saving) {
      return;
    }
    dragged = task;
    startOrder = taskIds(list).join(",");
    task.classList.add("task-dragging");
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", task.dataset.taskId);
  });

  document.addEventListener("dragover", function (event) {
    if (!dragged) {
      return;
    }
    var target = event.target.closest && event.target.closest(".task");
    if (!target || target === dragged || target.parentNode !== dragged.parentNode) {
      return;
    }
    event.preventDefault();
    var rect = target.getBoundingClientRect();
    var after = event.clientY > rect.top + rect.height / 2;
    target.parentNode.insertBefore(dragged, after ? target.nextSibling : target);
  });

  document.addEventListener("drop", function (event) {
    if (dragged) {
      event.preventDefault();
    }
  });

  document.addEventListener("dragend", function () {
    if (!dragged) {
      return;
    }
    var task = dragged;
    dragged = null;
    task.classList.remove("task-dragging");
    task.removeAttribute("draggable");
    if (list && taskIds(list).join(",") !== startOrder) {
      announce(task);
      save(list);
    }
  });

  document.addEventListener("keydown", function (event) {
    var handle = event.target.closest && event.target.closest("[data-drag-handle]");
    if (!handle || !event.altKey || saving) {
      return;
    }
    var up = event.key === "ArrowUp";
    if (!up && event.key !== "ArrowDown") {
      return;
    }
    event.preventDefault();

    var task = handle.closest(".task");
    list = currentList();
    var sibling = up ? task.previousElementSibling : task.nextElementSibling;
    if (!sibling) {
      return;
    }
    list.insertBefore(task, up ? sibling : sibling.nextSibling);
    handle.focus();
    announce(task);
    save(list);
  });
})();
