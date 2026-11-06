# Мутационное тестирование (со звёздочкой)

Инструмент: [`gremlins`](https://github.com/go-gremlins/gremlins) v0.6.0,
`gremlins unleash <pkg> --workers=2 --timeout-coefficient=10`. Пакеты:
`internal/domain/trip`, `internal/usecase/trip/{create,get,finish}`.

## До (тесты, написанные для покрытия, без мишени на мутантов)

| Пакет | Killed | Lived | Not covered | Efficacy |
|---|---|---|---|---|
| `domain/trip` | 16 | 5 | 0 | 76.2% |
| `usecase/trip/create` | 6 | 0 | 1 | 100.0%* |
| `usecase/trip/get` | — | — | — | нет мутируемых конструкций |
| `usecase/trip/finish` | 3 | 0 | 0 | 100.0% |

\* формально 100%, но один мутант вообще не попадал под тест (`Not covered`),
то есть по нему тесты и не запускались.

Все 5 выживших в `domain/trip` — `CONDITIONALS_BOUNDARY` на границах валидации:

```
LIVED CONDITIONALS_BOUNDARY at trip.go:45:11   price < 0        →  price <= 0
LIVED CONDITIONALS_BOUNDARY at trip.go:62:16   c.Latitude < -90   → c.Latitude <= -90
LIVED CONDITIONALS_BOUNDARY at trip.go:62:36   c.Latitude > 90    → c.Latitude >= 90
LIVED CONDITIONALS_BOUNDARY at trip.go:66:17   c.Longitude < -180 → c.Longitude <= -180
LIVED CONDITIONALS_BOUNDARY at trip.go:66:39   c.Longitude > 180  → c.Longitude >= 180
```

Покрытие по строкам (`go test -cover`) было 95.5% — эти пять строк
выполнялись в тестах (обычные невалидные и валидные значения), но ни один
тест не стоял ровно на границе. Мутация сдвигает `<` в `<=`: `price == 0`
стал бы ошибочно отклонён, `latitude == -90`/`90` и `longitude == -180`/`180`
(легальные края диапазона) — тоже, а тесты этого не заметили бы.

`Not covered` в `usecase/trip/create` (`usecase.go:69:80`) — проверка ошибки
`CompleteIdempotencyKey` вообще не выполнялась ни в одном тесте: существующие
тесты либо не передавали `IdempotencyKey`, либо уходили в более ранний return
(конфликт/повтор), не доходя до этой строки.

## Разбор и правки

- `internal/domain/trip/trip_test.go`: добавлены 5 кейсов на точные границы —
  `price = 0`, `latitude = ±90`, `longitude = ±180` — как валидные значения.
- `internal/usecase/trip/create/usecase_test.go`: добавлены
  `TestUsecase_Execute_Create_NewIdempotencyKey` (счастливый путь с новым
  ключом, доходит до `CompleteIdempotencyKey`) и
  `TestUsecase_Execute_CompleteIdempotencyKeyError` (тот же путь, но
  `CompleteIdempotencyKey` возвращает ошибку).

`usecase/trip/get` не дал ни одного мутанта — в `Execute` нет ветвлений,
арифметики или сравнений (один вызов `context.WithTimeout` и прямой `return`
результата репозитория), мутаторам `gremlins` просто не на чем сработать. Это
не пробел в тестах, а отсутствие мутируемого кода.

## После

| Пакет | Killed | Lived | Not covered | Efficacy |
|---|---|---|---|---|
| `domain/trip` | 21 | 0 | 0 | **100%** |
| `usecase/trip/create` | 7 | 0 | 0 | **100%** |
| `usecase/trip/get` | — | — | — | нет мутируемых конструкций |
| `usecase/trip/finish` | 3 | 0 | 0 | **100%** |

Выживших не осталось — делать вид, что это абсолютная метрика, не стоит:
`gremlins` по умолчанию проверяет только пять видов мутаций
(`ARITHMETIC_BASE`, `CONDITIONALS_BOUNDARY`, `CONDITIONALS_NEGATION`,
`INCREMENT_DECREMENT`, `INVERT_NEGATIVES`), остальные (`INVERT_LOGICAL`,
`INVERT_LOOPCTRL` и т.д.) выключены по умолчанию и не запускались. 100% здесь
значит «тесты ловят любое изменение сравнения или арифметики в этих пяти
файлах», не более и не менее.
