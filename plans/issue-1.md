<!-- adw:plan -->
### ADW · plan (revisión 1)
Recoge: la especificación aclarada. Sin peticiones de cambio.

# Feature: reordenar tareas con drag and drop y animación profesional

Issue: https://github.com/mats-romero/desarrollo-agentico-todo-04/issues/1 — «Quiero poder reordenar tareas con drag and drop y que la animación sea profesional».

Rama de trabajo: `feature/issue-1-reordenar-tareas-drag-and-drop` (al planificar, igual que `main` y sin cambios locales).

## Encargo y criterios de aceptación

### Requisitos explícitos (respuestas del humano)

- R1. El orden se guarda en el servidor: se mantiene al recargar y en otros navegadores/dispositivos.
- R2. Pendientes y completadas se muestran en bloques separados; se reordena **dentro** de cada bloque.
- R3. Una tarea nueva aparece al principio de la lista (de pendientes).
- R4. Animación: al arrastrar, la tarea se «levanta» con sombra; las demás se desplazan suavemente para hacerle hueco; al soltar, encaja con una transición corta.
- R5. Solo ratón. No hace falta soporte táctil ni de teclado.

### Inferencias y decisiones técnicas (no pedidas literalmente)

- I1. El bloque de pendientes va primero y el de completadas después, cada uno con un título visible («Pendientes», «Completadas»). Un bloque vacío no muestra su título.
- I2. No se puede arrastrar una tarea de un bloque a otro: el cambio de bloque sigue haciéndose con el botón de completar/reabrir.
- I3. Al completar una tarea pasa al **principio** del bloque de completadas; al reabrirla pasa al **principio** del bloque de pendientes (coherente con R3: lo último que «entra» en un bloque aparece arriba).
- I4. Las tareas existentes conservan, dentro de su bloque, el orden que tienen hoy (más antigua primero).
- I5. Toda la fila es el asa de arrastre (cursor `grab`), salvo los botones, enlaces y formularios, que siguen funcionando con clic. El arrastre empieza tras mover el ratón 4 px, para no interferir con los clics.
- I6. Implementación propia en JavaScript sin librerías (pointer events + transforms), siguiendo el estilo de `public/assets/app.js`. No se añaden dependencias.
- I7. Reordenar no cambia `updated_at` (no es una edición del contenido).
- I8. Si el servidor rechaza un reordenamiento (p. ej. otra pestaña cambió la lista), la página se recarga para mostrar el orden real.
- I9. Con `prefers-reduced-motion: reduce` el reordenamiento funciona igual pero sin animaciones (la regla CSS global existente ya anula las transiciones).
- I10. Fuera de alcance: autoscroll al arrastrar cerca del borde de la ventana (las listas son cortas).

### Criterios de aceptación

1. La página `/` muestra dos bloques: «Pendientes» (tareas con `completed = false`) y, debajo, «Completadas». Cada bloque está ordenado por la posición guardada.
2. Al arrastrar con el ratón una tarea dentro de su bloque: la fila se eleva (sombra y ligero escalado), sigue el puntero en vertical sin salir de su bloque, y las demás filas se desplazan con transición suave para abrir hueco.
3. Al soltar, la fila se desliza hasta su hueco con una transición corta (~180 ms) y la sombra desaparece; el DOM queda en el nuevo orden sin saltos.
4. El nuevo orden se envía al servidor (`POST /tasks/reorder`) y, al recargar o abrir la app en otro navegador, se ve el mismo orden.
5. No se puede soltar una tarea en el otro bloque.
6. Al crear una tarea, aparece la primera del bloque de pendientes.
7. Al completar/reabrir una tarea (acción `PUT /tasks/{id}/toggle`), el servidor la coloca la primera de su bloque de destino; en la página, el JS la mueve al principio del otro bloque.
8. Hacer clic en completar, editar o eliminar sigue funcionando igual; un clic sin movimiento no inicia arrastre.
9. Editar el título o eliminar una tarea no altera el orden de las demás.
10. `POST /tasks/reorder` responde 204 si la lista de ids coincide exactamente con las tareas del bloque; 422 si el bloque o algún id no es válido o hay ids repetidos; 409 si el conjunto de ids no coincide con el del bloque en la base de datos. En 422/409 no se modifica ninguna posición.

## Comportamiento actual y propuesta

**Actual** (verificado en el código):

- `migrations/20260901000000_create_tasks.up.fizz`: tabla `tasks` con `id`, `title`, `completed`, timestamps. No hay columna de orden.
- `actions/tasks.go`: `TasksIndex` y la rama de error de `TasksCreate` cargan `tx.Order("created_at asc, id asc").All(&tasks)` (consulta duplicada). Una sola lista mezcla pendientes y completadas; las nuevas aparecen al final.
- `templates/tasks/index.plush.html`: un único `<ul class="task-list">` con un `<li class="task">` por tarea.
- `public/assets/app.js`: intercepta el submit de `[data-task-toggle]`, actualiza la fila en el sitio (clase `task-completed`, etiquetas ARIA, `.task-status`) y lanza `fetch(form.action)`.
- No hay JavaScript de arrastre ni librerías JS; `app.js` y `app.css` se sirven desde `public/assets` con un parámetro `?v=N` de cache-busting en `templates/application.plush.html`.

**Propuesta** (cambio vertical mínimo):

- Nueva columna `position integer not null default 0` en `tasks`. El orden de un bloque es `position asc, created_at desc, id asc` (el desempate hace que, ante posiciones iguales, la más reciente quede arriba).
- Las inserciones al principio de un bloque (crear, completar, reabrir) usan `MIN(position) - 1` del bloque de destino. Los enteros negativos son válidos; no hace falta renumerar.
- Reordenar reescribe las posiciones del bloque completo como `0..n-1` a partir de la lista ordenada de ids que envía el cliente.
- La plantilla pinta dos listas (`data-task-list="pending"` / `"completed"`) con un partial común para la fila.
- `app.js` añade el arrastre con ratón y la animación, y hace que el toggle mueva la fila al principio del otro bloque.

## Alcance

**Incluido**

- Migración que añade y rellena `position`.
- Campo `Position` en `models.Task` y helpers de modelo para cargar la lista, calcular la posición superior de un bloque y reordenar un bloque.
- Ruta y acción `POST /tasks/reorder`.
- Posicionamiento en `TasksCreate` y `TasksToggle`.
- Plantilla en dos bloques con partial de fila.
- JS de drag and drop con animación y CSS asociado; movimiento de bloque en el toggle del lado del cliente.
- Tests de modelo y de acción.

**Excluido**

- Soporte táctil y de teclado (R5).
- Arrastrar entre bloques (I2).
- Autoscroll durante el arrastre (I10).
- Librerías JS de terceros (I6).
- Tests de navegador/E2E automatizados (el repo no tiene infraestructura para ello); el JS se valida manualmente.
- Corregir el problema de persistencia del toggle descrito en «Observaciones».

## Diseño por capas

### Persistencia y migración

Nuevo par de ficheros fizz `migrations/20261014000000_add_position_to_tasks.{up,down}.fizz` (el `//go:embed *.fizz *.sql` de `migrations/embed.go` ya los incluye; `models.Migrate()` los aplica al arrancar la app y `soda migrate -e test` en CI).

Up:

```
add_column("tasks", "position", "integer", {"default": 0})

sql("UPDATE tasks SET position = ranked.rn - 1 FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY completed ORDER BY created_at ASC, id ASC) AS rn FROM tasks) AS ranked WHERE tasks.id = ranked.id")
```

Down:

```
drop_column("tasks", "position")
```

(En fizz, `add_column` sin `"null": true` crea la columna `NOT NULL`; con el `default` 0 las filas existentes, el seed `20260901000001_seed_tasks.up.sql` y los fixtures sin `position` siguen siendo válidos.) No se añade índice: las listas son pequeñas.

### Modelo (`models/task.go`)

- Añadir a `Task`: `Position int `json:"position" db:"position" form:"-"`` (el `form:"-"` evita que `c.Bind` en `TasksCreate` lo acepte del formulario, igual que `Completed`).
- Constante `taskListOrder = "completed asc, position asc, created_at desc, id asc"`.
- `func LoadTasks(tx *pop.Connection) (Tasks, error)`: `tx.Order(taskListOrder).All(&tasks)`.
- `func (ts Tasks) Pending() Tasks` y `func (ts Tasks) Completed() Tasks`: filtran conservando el orden.
- `func TopPosition(tx *pop.Connection, completed bool) (int, error)`: devuelve `COALESCE(MIN(position), 0) - 1` del bloque. Implementación sugerida: `tx.RawQuery("SELECT COALESCE(MIN(position), 0) - 1 AS position FROM tasks WHERE completed = ?", completed).First(&row)` con `row := struct{ Position int `db:"position"` }{}`.
- `var ErrReorderMismatch = errors.New(...)` y `func ReorderTasks(tx *pop.Connection, completed bool, ids []uuid.UUID) error`:
  1. Carga los ids del bloque (`tx.Where("completed = ?", completed).All(&tasks)` o `RawQuery` de ids).
  2. Si `len(ids)` difiere, hay ids repetidos o algún id no pertenece al bloque → `ErrReorderMismatch` sin escribir nada.
  3. Para cada `i, id`: `tx.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", i, id).Exec()` (no toca `updated_at`, I7).
  La detección de repetidos puede hacerse en la acción (422) antes de llamar; dentro de `ReorderTasks` basta con comprobar que el conjunto coincide, tratando repetidos como mismatch por seguridad.

### Acción y ruta (`actions/tasks.go`, `actions/app.go`)

- `TasksIndex`: usar `models.LoadTasks(tx)` y `c.Set("tasks", tasks)`, `c.Set("pendingTasks", tasks.Pending())`, `c.Set("completedTasks", tasks.Completed())`. Extraer un helper privado `setTaskList(c buffalo.Context, tx *pop.Connection) error` para reutilizarlo en la rama de error de `TasksCreate` (hoy duplica la consulta).
- `TasksCreate`: tras normalizar el título y antes de `ValidateAndCreate`, `task.Position, err = models.TopPosition(tx, false)`. (El cálculo puede hacerse antes de validar; si la validación falla no se inserta nada.)
- `TasksToggle`: tras invertir `Completed`, `task.Position, err = models.TopPosition(tx, task.Completed)` y después `tx.Update(task)`.
- `TasksUpdate` y `TasksDestroy`: sin cambios (Update recarga la tarea con `Find`, así que conserva `Position`; borrar deja huecos inocuos).
- Nueva `TasksReorder(c buffalo.Context) error`:
  1. `req := c.Request(); req.ParseForm()`; `block := req.PostForm.Get("block")`; `rawIDs := req.PostForm["ids"]`.
  2. `block` debe ser `pending` o `completed` → si no, `c.Error(http.StatusUnprocessableEntity, ...)`.
  3. Parsear cada id con `uuid.FromString`; si alguno falla o hay repetidos → 422.
  4. `models.ReorderTasks(tx, block == "completed", ids)`; si `errors.Is(err, models.ErrReorderMismatch)` → `c.Error(http.StatusConflict, err)`; otro error → devolverlo.
  5. Éxito → `c.Render(http.StatusNoContent, nil)`.
- Ruta en `App()`: `app.POST("/tasks/reorder", TasksReorder)`. Se usa POST (y no `PUT /tasks/reorder`) para no competir con el patrón `PUT /tasks/{task_id}` y no depender del orden de registro de rutas.
- CSRF: el middleware `csrf.New` sigue activo; el cliente envía `authenticity_token` en el cuerpo y en la cabecera `X-CSRF-Token`. En el entorno de test el middleware no bloquea (los tests actuales hacen POST/PUT sin token).

### Renderizado (`templates/tasks/`)

- Nuevo partial `templates/tasks/_task.plush.html` con el `<li>` actual (sin cambios de contenido) más `data-task-id="<%= item.ID %>"`. Usar la variable `item` (no `task`, que ya es el modelo del formulario de creación). Se invoca con `<%= partial("tasks/task.plush.html", {item: t}) %>`, siguiendo el patrón de `partial("flash.plush.html")`. Sustituir `t.` por `item.` dentro del partial.
- `templates/tasks/index.plush.html`: en la rama con tareas, sustituir el `<ul>` único por:

```
<div class="task-groups" data-task-groups data-reorder-url="/tasks/reorder" data-csrf-token="<%= authenticity_token %>">
  <section class="task-group" aria-labelledby="pending-heading">
    <h3 id="pending-heading" class="task-group-title">Pendientes</h3>
    <ul class="task-list" data-task-list="pending">
      <%= for (t) in pendingTasks { %><%= partial("tasks/task.plush.html", {item: t}) %><% } %>
    </ul>
  </section>
  <section class="task-group" aria-labelledby="completed-heading">
    <h3 id="completed-heading" class="task-group-title">Completadas</h3>
    <ul class="task-list" data-task-list="completed">
      <%= for (t) in completedTasks { %><%= partial("tasks/task.plush.html", {item: t}) %><% } %>
    </ul>
  </section>
</div>
```

  Ambos bloques se pintan siempre que haya tareas (el JS del toggle necesita la lista de destino); el CSS oculta el bloque vacío (I1). El estado vacío y el contador `len(tasks)` no cambian.
- `templates/application.plush.html`: subir el cache-busting a `app.css?v=4` y `app.js?v=3`.

### JavaScript (`public/assets/app.js`)

Mismo estilo que el fichero actual (ES5: `var`, `function`, sin módulos). Dos cambios:

**A. Toggle**: tras actualizar la fila, moverla al principio de la lista de destino:
`var target = document.querySelector('[data-task-list="' + (completed ? "completed" : "pending") + '"]'); if (target) target.insertBefore(task, target.firstElementChild);`

**B. Drag and drop** (envuelto en una IIFE; estado en un objeto `drag` o `null`):

- `pointerdown` delegado en `document` sobre `.task-list .task`:
  - ignorar si `event.pointerType !== "mouse"` o `event.button !== 0` (R5);
  - ignorar si `event.target.closest("button, a, input, select, textarea")` (los `<form>` de la fila solo contienen botones e inputs ocultos, así que basta con esos selectores);
  - `event.preventDefault()` (evita selección de texto) y guardar: `item`, `list`, `pointerId`, `startY = event.clientY`, `started = false`.
- `pointermove` (en `document`, solo si hay `drag` y coincide `pointerId`):
  - si `!started` y `Math.abs(dy) < 4` → nada (I5). Al superar el umbral, **iniciar**:
    - `item.setPointerCapture(pointerId)`;
    - cachear `siblings` = los `.task` de `list` en orden DOM excluyendo `item`, `startIndex` = índice del item en la lista, `itemRect`, `itemHeight = item.offsetHeight`, y para cada sibling su `center` y `height` iniciales (los rects se cachean porque luego cambian por los transforms);
    - límites de `dy`: `minDy = listRect.top - itemRect.top`, `maxDy = listRect.bottom - itemRect.bottom` (no sale de su bloque, criterio 5);
    - clases: `item` → `is-dragging`; `list` → `is-sorting`; `document.body` → `is-dragging-task`.
  - en cada movimiento: `dy` acotado; `item.style.transform = "translateY(" + dy + "px) scale(1.02)"`;
  - `targetIndex` = número de siblings cuyo `center` inicial es menor que `itemCenterInicial + dy`;
  - para cada sibling `i`: si `i < startIndex && i >= targetIndex` → `translateY(itemHeight px)`; si `i >= startIndex && i < targetIndex` → `translateY(-itemHeight px)`; si no → `""`. Las transiciones CSS producen el desplazamiento suave (R4). Usar `requestAnimationFrame` para agrupar escrituras.
- `pointerup`:
  - si `!started` → limpiar estado y no hacer nada (el clic sigue su curso, criterio 8);
  - si `started`: calcular el destino del item: si `targetIndex > startIndex`, `offset = suma de height de siblings[startIndex .. targetIndex-1]`; si `targetIndex < startIndex`, `offset = -suma de height de siblings[targetIndex .. startIndex-1]`; si no, 0. Añadir `is-dropping` al item y fijar `transform = "translateY(" + offset + "px)"` (escala 1, la sombra se desvanece por transición). Al terminar (`transitionend` del item filtrando `propertyName === "transform"`, **más** un `setTimeout` de respaldo de ~250 ms, y sin esperar si `matchMedia("(prefers-reduced-motion: reduce)").matches`, porque la regla CSS global anula las transiciones y `transitionend` no llegaría) ejecutar `finish()` una sola vez:
    - quitar `is-sorting` de la lista (desactiva transiciones de siblings) y limpiar `style.transform` de item y siblings;
    - `list.insertBefore(item, siblings[targetIndex] || null)`;
    - forzar reflow (`void list.offsetHeight`) y quitar `is-dragging`/`is-dropping`/`is-dragging-task`;
    - si `targetIndex !== startIndex` → `persist(list)`.
  - suprimir el `click` que el navegador pueda emitir tras el arrastre: listener de `click` en fase de captura, de un solo uso, que haga `preventDefault()` + `stopPropagation()`, retirado en el siguiente `setTimeout(…, 0)`.
- `pointercancel`, `keydown` Escape y `blur` de `window` durante un arrastre → cancelar: animar item y siblings a `translateY(0)` y limpiar igual que `finish()` sin mover ni persistir.
- `persist(list)`:
  - `groups = document.querySelector("[data-task-groups]")`; `ids` = `data-task-id` de los `.task` de `list` en orden;
  - `body = new URLSearchParams()` con `authenticity_token`, `block = list.dataset.taskList` y un `ids` por id;
  - `fetch(groups.dataset.reorderUrl, { method: "POST", body: body, credentials: "same-origin", headers: { "X-CSRF-Token": token } })`; si `!response.ok` o hay error de red → `window.location.reload()` (I8).
- Solo un arrastre a la vez; ignorar `pointerdown` mientras hay uno en curso o en `finish` pendiente.

### CSS (`public/assets/app.css`)

- `.task-group + .task-group { margin-top: 1.75rem; }` y `.task-group-title` (texto pequeño, `font-weight: 600`, color `var(--color-muted)`, margen inferior).
- Ocultar bloque vacío: `.task-group:not(:has(.task)) { display: none; }`.
- `.task-list .task { position: relative; background: #ffffff; cursor: grab; }` solo bajo `@media (hover: hover) and (pointer: fine)` (ya existe ese bloque: añadir ahí el `cursor`).
- `.task-list.is-sorting .task:not(.is-dragging) { transition: transform 200ms cubic-bezier(0.2, 0, 0, 1); }`.
- `.task.is-dragging { z-index: 10; cursor: grabbing; border-radius: 8px; border-bottom-color: transparent; box-shadow: 0 12px 28px rgba(40, 40, 38, 0.18), 0 2px 6px rgba(40, 40, 38, 0.10); transition: box-shadow 150ms ease; will-change: transform; }` (sin transición de `transform` mientras sigue al ratón).
- `.task.is-dragging.is-dropping { transition: transform 180ms cubic-bezier(0.2, 0, 0, 1), box-shadow 180ms ease; box-shadow: none; }`.
- `body.is-dragging-task { cursor: grabbing; user-select: none; -webkit-user-select: none; }` y `body.is-dragging-task .task-actions { opacity: 0; }` (evita que los iconos parpadeen por `:hover` al pasar sobre otras filas).
- La regla existente `@media (prefers-reduced-motion: reduce)` ya anula todas las transiciones; no añadir nada (I9).

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `migrations/20261014000000_add_position_to_tasks.up.fizz` (nuevo) | Añade `position` y la rellena por bloque conservando el orden actual |
| `migrations/20261014000000_add_position_to_tasks.down.fizz` (nuevo) | Elimina `position` |
| `models/task.go` | Campo `Position`, `LoadTasks`, `Pending`/`Completed`, `TopPosition`, `ReorderTasks`, `ErrReorderMismatch` |
| `models/task_test.go` | Tests de modelo de los helpers anteriores |
| `actions/tasks.go` | `setTaskList`, posición en create/toggle, nueva `TasksReorder` |
| `actions/app.go` | Ruta `POST /tasks/reorder` |
| `actions/tasks_test.go` | Tests de acción de orden, creación, toggle y reorder |
| `templates/tasks/index.plush.html` | Dos bloques con `data-task-list` y datos para el JS |
| `templates/tasks/_task.plush.html` (nuevo) | Partial de la fila, con `data-task-id` |
| `templates/application.plush.html` | Cache-busting de `app.css` y `app.js` |
| `public/assets/app.js` | Drag and drop con animación, persistencia y toggle entre bloques |
| `public/assets/app.css` | Estilos de bloques, elevación, desplazamiento y caída |
| `fixtures/tasks.toml` | Sin cambios (las filas sin `position` toman el default 0) |

## Implementación paso a paso

1. Crear la migración up/down descrita en «Persistencia y migración».
2. En `models/task.go`, añadir `Position` y los helpers (`taskListOrder`, `LoadTasks`, `Pending`, `Completed`, `TopPosition`, `ErrReorderMismatch`, `ReorderTasks`). Importar `errors` y usar `uuid` (ya importado).
3. Añadir los tests de modelo (ver «Tests y validación») y ejecutarlos.
4. En `actions/tasks.go`: crear `setTaskList`; usarlo en `TasksIndex` y en la rama de error de `TasksCreate`; asignar `Position` en `TasksCreate` y `TasksToggle`; añadir `TasksReorder` (imports `errors`, `github.com/gofrs/uuid`).
5. En `actions/app.go`, registrar `app.POST("/tasks/reorder", TasksReorder)` junto a las demás rutas de tareas.
6. Crear `templates/tasks/_task.plush.html` moviendo el `<li>` de `index.plush.html` (renombrar `t` → `item`, añadir `data-task-id`), y reescribir la lista de `index.plush.html` con los dos bloques.
7. Subir versiones de assets en `templates/application.plush.html`.
8. Añadir los tests de acción y ejecutarlos.
9. Implementar en `public/assets/app.js` el cambio del toggle (A) y el drag and drop (B).
10. Añadir los estilos en `public/assets/app.css`.
11. Ejecutar la validación completa y la prueba manual.

## Tests y validación

### Tests de modelo (`models/task_test.go`)

- `Test_Task_TopPosition_EmptyBlock`: sin tareas, `TopPosition(tx, false)` = -1.
- `Test_Task_TopPosition_BelowMinimum`: con tareas pendientes en posiciones 0 y 3 y una completada en -10, `TopPosition(tx, false)` = -1 y `TopPosition(tx, true)` = -11.
- `Test_Task_LoadTasks_OrdersByBlockAndPosition`: crear tareas con posiciones/estados mezclados y comprobar que `LoadTasks` devuelve pendientes por `position` y luego completadas por `position`; y que `Pending()`/`Completed()` las separan.
- `Test_Task_ReorderTasks_RewritesPositions`: tres pendientes; reordenar `[c, a, b]` → posiciones 0,1,2 respectivamente; `updated_at` sin cambios.
- `Test_Task_ReorderTasks_Mismatch`: ids incompletos, con un id de otro bloque o con un id inexistente → `ErrReorderMismatch` y posiciones intactas.

### Tests de acción (`actions/tasks_test.go`)

- `Test_TasksIndex_SeparatesBlocks`: una completada con posición menor que dos pendientes; en el HTML, `strings.Index` de los títulos pendientes es menor que el de «Completadas» y que el de la tarea completada; aparecen `data-task-list="pending"` y `data-task-list="completed"`.
- `Test_TasksIndex_OrdersByPosition`: pendientes con posiciones 2, 0, 1 aparecen en el orden 0, 1, 2; incluye `data-task-id="<id>"`.
- `Test_TasksCreate_PlacesNewTaskFirst`: con una pendiente existente en posición 0, crear otra por POST → su `Position` es menor y aparece antes en `/`.
- `Test_TasksToggle_MovesToTopOfDestination`: con completadas en posiciones 0 y 1, completar una pendiente → su posición < 0; reabrirla → posición menor que la mínima de pendientes.
- `Test_TasksReorder_Persists`: tres pendientes; `as.HTML("/tasks/reorder").Post(url.Values{"block": {"pending"}, "ids": {c, a, b}})` → 204; recargar y comprobar posiciones 0,1,2 y orden en `GET /`.
- `Test_TasksReorder_InvalidBlock` (`block=otro`) → 422.
- `Test_TasksReorder_InvalidID` (`ids` con `no-es-un-uuid`) → 422.
- `Test_TasksReorder_DuplicateIDs` → 422.
- `Test_TasksReorder_Mismatch` (falta un id del bloque, o incluye una tarea completada en `block=pending`) → 409 y posiciones sin cambios.
- Los tests existentes (`Test_TasksIndex_ListsTasks`, `Test_TasksToggle_*`, `Test_TasksUpdate_*`, etc.) deben seguir pasando sin modificación. `Test_TasksUpdate_PersistsTitleOnly` puede ampliarse para comprobar que `Position` no cambia.

### Comandos

Requieren PostgreSQL (servicio `db` de `compose.yaml` en `127.0.0.1:5432`) y la base `todo_test` migrada:

```
GO_ENV=test soda migrate -e test
gofmt -l .
go vet ./...
GO_ENV=test go test -p 1 ./...
go build ./...
```

### Prueba manual (navegador de escritorio, con la app reconstruida en Docker para aplicar la migración)

1. Reordenar pendientes arrastrando con ratón: comprobar elevación con sombra, desplazamiento suave de las demás, encaje al soltar; recargar → el orden se conserva; abrir en otro navegador → mismo orden.
2. Igual en el bloque de completadas; intentar llevar una tarea al otro bloque → se queda en el límite de su bloque.
3. Clic en completar/editar/eliminar sigue funcionando; un clic sin moverse no arrastra.
4. Crear una tarea → aparece la primera de pendientes.
5. Escape o soltar fuera de la ventana durante un arrastre → la fila vuelve a su sitio.
6. Con «reducir movimiento» activado en el SO → reordenar funciona sin animaciones y sin quedarse bloqueado.
7. Dos pestañas: en una completar una tarea (recargando), en la otra reordenar pendientes sin recargar → la petición devuelve 409 y la página se recarga con el orden real.

## Riesgos y recuperación

- **Migración en datos reales**: el `UPDATE` con `ROW_NUMBER()` es idempotente respecto al resultado y la migración down elimina la columna. Recuperación: `soda migrate down` o revertir el commit y reconstruir.
- **Rutas**: `POST /tasks/reorder` no colisiona con ninguna ruta existente (no hay `POST /tasks/{id}`).
- **CSRF en `fetch`**: si el middleware rechazara la petición (403), el JS recarga la página y el orden visual vuelve al del servidor; revisar que se envían `authenticity_token` y la cabecera `X-CSRF-Token`.
- **Posiciones negativas crecientes**: crear/completar decrementa la posición mínima; con `int` de 32 bits no hay riesgo práctico. Posiciones repetidas por concurrencia se desempatan con `created_at desc, id asc`.
- **Carreras entre pestañas/dispositivos**: un reordenamiento con un conjunto de ids desactualizado devuelve 409 y el cliente recarga (I8); nunca se aplican posiciones parciales (la acción corre dentro de la transacción de `popmw.Transaction`).
- **`transitionend` que no llega** (reduced motion, pestaña en segundo plano): mitigado con el `setTimeout` de respaldo y la comprobación de `matchMedia`.
- **`:has()`** para ocultar bloques vacíos: soportado por navegadores actuales; si no, solo se vería un título sin filas.
- **Filas de alturas distintas** (títulos largos): el desplazamiento usa la altura real de la fila arrastrada y la suma de alturas de las filas superadas, así que el encaje es exacto.

## Observaciones (fuera del encargo, no investigadas)

- `public/assets/app.js` hace `fetch(form.action)` al completar/reabrir: es una petición **GET** a `/tasks/{id}/toggle`, que no tiene ruta GET (la ruta es `PUT` con `_method` en el formulario) y no lleva el token CSRF. Aparentemente el cambio de estado desde la interfaz con JS no se guarda en el servidor y se pierde al recargar. Afecta a la experiencia de esta feature (la tarea se mueve de bloque en la página pero, al recargar, volvería a su estado anterior), pero este plan no lo corrige; conviene abrir una issue aparte. Los tests de servidor del toggle (`PUT`) no se ven afectados.
