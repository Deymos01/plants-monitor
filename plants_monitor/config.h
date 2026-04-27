#ifndef CONFIG_H
#define CONFIG_H

// =====================
// Pins
// =====================
const int LDR_PIN = 32;
const int SOIL_PIN = 33;

// =====================
// ADC settings
// =====================
const int SAMPLES = 50;

// =====================
// Moisture interval
// =====================
const int SOIL_DRY_RAW = 0;
const int SOIL_WET_RAW = 2200;

// =====================
// Deep sleep
// =====================
// 2 h = 7200 s
// const uint64_t DEFAULT_SLEEP_SECONDS = 7200ULL;
const uint64_t DEFAULT_SLEEP_SECONDS = 10ULL;

#endif