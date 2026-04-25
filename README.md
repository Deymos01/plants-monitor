# Plant Monitor

Plant Monitor — это IoT-проект для мониторинга состояния домашнего растения. Устройство на ESP32 измеряет влажность
почвы и уровень освещённости, после чего отправляет данные на Go-сервер. Сервер принимает измерения по HTTP API,
авторизует устройство по токену и сохраняет данные в SQLite.

## Возможности

- измерение влажности почвы;
- измерение освещённости через фоторезистор;
- отправка данных с ESP32 на сервер по HTTP;
- авторизация устройства через `device_id` и `device_token`;
- хранение измерений в SQLite;
- получение последнего измерения устройства через API;

## Принцип работы

ESP32 подключается к Wi-Fi, считывает значения с датчиков, формирует JSON и отправляет его на Go-сервер через
`POST /api/v1/measurements`. После чего уходит в глубокий сон на 2 часа.

Сервер проверяет заголовки:

```http
X-Device-ID: ...
X-Device-Token: ...
```

Если устройство авторизовано, измерение сохраняется в SQLite.

## Требования

Для серверной части:

- Go 1.26;
- доступный TCP-порт, например `8080`.

Для ESP32:

- ESP32 DevKit или совместимая плата;
- Arduino IDE или Arduino CLI;
- Датчик влажности почвы с аналоговым выходом;
- Фоторезистор;
- Резистор для делителя напряжения фоторезистора;
- Wi-Fi сеть 2.4 GHz.

## Настройка Go-сервера

### 1. Создание .env файла

Создайте файл `.env` в корне проекта и добавьте следующие переменные:

```env
HTTP_ADDR=0.0.0.0:8080
DB_PATH=data/plant_monitor.db
```

Описание переменных:

| Переменная  | Назначение                                    |
|-------------|-----------------------------------------------|
| `HTTP_ADDR` | Адрес и порт, на котором будет запущен сервер |
| `DB_PATH`   | Путь к SQLite базе данных                     |

### 2. Запуск сервера

1. Установите зависимости:

```bash
go mod tidy
```

2. Запустите сервер:

```bash
go run ./cmd/server
```

## Регистрация устройства

Перед тем как ESP32 сможет отправлять измерения, устройство нужно зарегистрировать на сервере.

### 1. Получить `device_id`

При запуске прошивки ESP32 печатает свой device_id в Serial Monitor, например:

```
Device ID: plant-a4cf12345678
```

### 2. Создать устройство на сервере

Отправьте POST-запрос на сервер с данным `device_id` для его регистрации:

```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "plant-a4cf12345678",
    "plant_name": "Фиалки"
  }'
```

Сервер вернёт `device_token`:

```json
{
  "ok": true,
  "device_id": "plant-a4cf12345678",
  "plant_name": "Фиалки",
  "device_token": "pmon_..."
}
```

Необходимо сохранить `device_token`. Он показывается только при создании устройства один раз и пригодится на следующем
шаге!

## Настройка ESP32

### 1. Создать `secrets.h`

В папке со скетчем ESP32 нужно скопировать шаблон секрета:

```bash
cp plants_monitor/secrets.example.h plants_monitor/secrets.h
```

И заполнить `secrets.h` своими данными:

```cpp
#ifndef SECRETS_H
#define SECRETS_H

const char* WIFI_SSID = "YOUR_WIFI_SSID";
const char* WIFI_PASSWORD = "YOUR_WIFI_PASSWORD";

const char* SERVER_URL = "http://192.168.1.34:8080/api/v1/measurements";

const char* DEVICE_TOKEN = "pmon_YOUR_DEVICE_TOKEN";

#endif
```

`SERVER_URL` должен указывать на IP-адрес машины, где запущен Go-сервер.

### 2. Проверить config.h и подключить датчики

В `config.h` проверьте, что пины для датчиков указаны правильно. Подключите датчики к ESP32 согласно этим пинам.

### 3. Загрузить прошивку

Откройте `plants_monitor/plants_monitor.ino` в Arduino IDE, выберите плату ESP32 и загрузите скетч.

Если всё настроено правильно, ESP32:

1. напечатает свой device_id;
2. считает датчики;
3. подключится к Wi-Fi;
4. отправит измерение на сервер;
5. выведет HTTP-статус ответа;
6. заснет на 2 часа.

## Формат данных от ESP32

ESP32 отправляет JSON:

```json
{
  "soil_raw": 1200.0,
  "soil_voltage": 1.245,
  "soil_percent": 55,
  "light_raw": 1800.0,
  "light_voltage": 1.620
}
```

Авторизация передаётся в HTTP-заголовках:

```http
X-Device-ID: plant-a4cf12345678
X-Device-Token: pmon_...
```

## API

### Создать устройство

```http request
POST /api/v1/devices
```

Пример:

```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "plant-a4cf12345678",
    "plant_name": "Фиалки"
  }'
```

### Отправить измерение

```http request
POST /api/v1/measurements
```

Пример:

```bash
curl -X POST http://localhost:8080/api/v1/measurements \
  -H "Content-Type: application/json" \
  -H "X-Device-ID: plant-a4cf12345678" \
  -H "X-Device-Token: pmon_YOUR_DEVICE_TOKEN" \
  -d '{
    "soil_raw": 1200,
    "soil_voltage": 1.245,
    "soil_percent": 55,
    "light_raw": 1800,
    "light_voltage": 1.620
  }'
```

### Получить последнее измерение устройства

```http request
GET /api/v1/measurements/latest?device_id=plant-a4cf12345678
```