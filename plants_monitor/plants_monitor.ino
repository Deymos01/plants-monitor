#include <WiFi.h>
#include <HTTPClient.h>

#include "config.h"
#include "secrets.h"

struct SensorReading {
  float raw;
  float voltage;
};

struct SensorData {
  SensorReading ldr;
  SensorReading soil;
  int soilPercent;
};

String getDeviceID() {
  uint64_t mac = ESP.getEfuseMac();

  char id[32];
  snprintf(
    id,
    sizeof(id),
    "plant-%04X%08X",
    (uint16_t)(mac >> 32),
    (uint32_t)mac
  );

  return String(id);
}

SensorReading readSensor(int pin) {
  uint32_t rawSum = 0;
  uint32_t mvSum = 0;

  for (int i = 0; i < SAMPLES; i++) {
    rawSum += analogRead(pin);
    mvSum += analogReadMilliVolts(pin);
    delay(5);
  }

  SensorReading result;
  result.raw = rawSum / (float)SAMPLES;
  result.voltage = mvSum / (float)SAMPLES / 1000.0;

  return result;
}

int soilMoisturePercent(float raw) {
  int percent = map((int)raw, SOIL_DRY_RAW, SOIL_WET_RAW, 0, 100);
  return constrain(percent, 0, 100);
}

SensorData readAllSensors() {
  SensorData data;

  data.ldr = readSensor(LDR_PIN);
  data.soil = readSensor(SOIL_PIN);
  data.soilPercent = soilMoisturePercent(data.soil.raw);

  return data;
}

bool connectWiFi(unsigned long timeoutMs = 20000) {
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  Serial.print("Connecting to WiFi");

  unsigned long start = millis();

  while (WiFi.status() != WL_CONNECTED && millis() - start < timeoutMs) {
    delay(500);
    Serial.print(".");
  }

  Serial.println();

  if (WiFi.status() == WL_CONNECTED) {
    Serial.println("WiFi connected");
    Serial.print("IP: ");
    Serial.println(WiFi.localIP());
    Serial.print("RSSI: ");
    Serial.println(WiFi.RSSI());
    return true;
  }

  Serial.println("WiFi connection failed");
  return false;
}

String buildMeasurementJSON(const SensorData& data) {
  String json;
  json.reserve(256);

  json += "{";

  json += "\"soil_raw\":" + String(data.soil.raw, 1) + ",";
  json += "\"soil_voltage\":" + String(data.soil.voltage, 3) + ",";
  json += "\"soil_percent\":" + String(data.soilPercent) + ",";

  json += "\"light_raw\":" + String(data.ldr.raw, 1) + ",";
  json += "\"light_voltage\":" + String(data.ldr.voltage, 3) + ",";

  json += "\"rssi\":" + String(WiFi.RSSI());

  json += "}";

  return json;
}

bool sendMeasurement(const String& deviceID, const SensorData& data) {
  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("Cannot send: WiFi is not connected");
    return false;
  }

  HTTPClient http;

  String json = buildMeasurementJSON(data);

  Serial.println("Sending measurement:");
  Serial.println(json);

  http.begin(SERVER_URL);

  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Device-ID", deviceID);
  http.addHeader("X-Device-Token", DEVICE_TOKEN);

  int httpCode = http.POST(json);

  Serial.print("HTTP status: ");
  Serial.println(httpCode);

  if (httpCode > 0) {
    String response = http.getString();

    Serial.println("Server response:");
    Serial.println(response);
  } else {
    Serial.print("HTTP error: ");
    Serial.println(http.errorToString(httpCode));
  }

  http.end();

  return httpCode >= 200 && httpCode < 300;
}

void goToSleep(uint64_t seconds) {
  Serial.print("Going to deep sleep for ");
  Serial.print(seconds);
  Serial.println(" seconds");

  Serial.flush();

  esp_sleep_enable_timer_wakeup(seconds * 1000000ULL);
  esp_deep_sleep_start();
}

void setup() {
  Serial.begin(115200);
  delay(1000);

  String deviceID = getDeviceID();

  Serial.println();
  Serial.println("ESP32 Plant Monitor");
  Serial.print("Device ID: ");
  Serial.println(deviceID);
  Serial.println();

  analogReadResolution(12);
  analogSetPinAttenuation(LDR_PIN, ADC_11db);
  analogSetPinAttenuation(SOIL_PIN, ADC_11db);

  SensorData data = readAllSensors();

  Serial.println("Sensor data:");

  Serial.print("LDR raw: ");
  Serial.print(data.ldr.raw, 1);
  Serial.print(" | voltage: ");
  Serial.print(data.ldr.voltage, 3);
  Serial.println(" V");

  Serial.print("Soil raw: ");
  Serial.print(data.soil.raw, 1);
  Serial.print(" | voltage: ");
  Serial.print(data.soil.voltage, 3);
  Serial.print(" V");
  Serial.print(" | percent: ");
  Serial.print(data.soilPercent);
  Serial.println("%");

  bool wifiOK = connectWiFi();

  if (wifiOK) {
    bool sent = sendMeasurement(deviceID, data);

    if (sent) {
      Serial.println("Measurement sent successfully");
    } else {
      Serial.println("Measurement send failed");
    }

    WiFi.disconnect(true);
    WiFi.mode(WIFI_OFF);
  }

  goToSleep(DEFAULT_SLEEP_SECONDS);
}

void loop() {}