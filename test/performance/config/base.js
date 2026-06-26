// ============================================================
// config/base.js
// Base URL, default headers, dan shared config
// ============================================================

export const BASE_URL =
    __ENV.BASE_URL || "https://aksacapturebe-production.up.railway.app";
export const API_BASE = `${BASE_URL}/api/v1`;

export const JSON_HEADERS = {
    headers: { "Content-Type": "application/json" },
};

// ─── Environment Variables ───────────────────────────────────
// Set via CLI: k6 run -e SKIP_R2=true test/performance/load_test.js
export const SKIP_R2 = true;
// export const SKIP_R2 = __ENV.SKIP_R2 === 'true';

// ─── Skenario nyata ──────────────────────────────────────────
// 5 user bersamaan, masing-masing 400 video
export const TOTAL_USERS = 5;
export const VIDEOS_PER_USER = 400;
export const TOTAL_VIDEOS = TOTAL_USERS * VIDEOS_PER_USER; // 2000

// ─── Data sample BISINDO ─────────────────────────────────────
export const LETTERS = [
    "A",
    "B",
    "C",
    "D",
    "E",
    "F",
    "G",
    "H",
    "I",
    "J",
    "K",
    "L",
    "M",
    "N",
    "O",
    "P",
    "Q",
    "R",
    "S",
    "T",
    "U",
    "V",
    "W",
    "X",
    "Y",
    "Z",
];

export const WORDS = [
    "selamat pagi",
    "selamat siang",
    "selamat sore",
    "selamat malam",
    "aku",
    "saya",
    "kamu",
    "dari",
    "mana",
    "berasal",
    "halo",
    "kabar",
    "apa",
    "siapa",
    "perkenalkan",
    "nama",
    "sayang",
    "marah",
];

export const SIGNERS = [
    { name: "Bintang", gender: "female" },
    { name: "Andi", gender: "male" },
    { name: "Siti", gender: "female" },
    { name: "Rizky", gender: "male" },
    { name: "Dewi", gender: "female" },
];

export const REGIONS = [
    { region: "Jawa Timur", subregion: "Malang" },
    { region: "Jawa Barat", subregion: "Bandung" },
    { region: "DKI Jakarta", subregion: "Jakarta" },
    { region: "Jawa Tengah", subregion: "Semarang" },
    { region: "Bali", subregion: "Denpasar" },
];

// ─── Default thresholds (dipakai di load & smoke) ────────────
export const DEFAULT_THRESHOLDS = {
    "http_req_duration{endpoint:upload_url}": ["p(95)<2000"],
    "http_req_duration{endpoint:save_metadata}": ["p(95)<5000"],
    http_req_failed: ["rate<0.02"],
};
