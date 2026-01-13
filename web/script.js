const API_URL = "/api/news";
const MARKET_URL = "/api/market";

let lastSeenId = "";
let audioEnabled = false;
let renderedIds = new Set();

document.addEventListener("DOMContentLoaded", () => {
    console.log("🚀 Crypto Intel Terminal v2.3 (Real-Time AI) Loaded");

    const toggle = document.getElementById("audio-toggle");
    if (toggle) {
        toggle.addEventListener("change", (e) => {
            audioEnabled = e.target.checked;
            if (audioEnabled) playSound("ping");
        });
    }

    fetchNews();
    fetchMarket();

    // Poll faster for updates so AI appears quickly
    setInterval(fetchNews, 2000);
    setInterval(fetchMarket, 10000);
});

async function fetchNews() {
    try {
        const res = await fetch(API_URL);
        if (!res.ok) return;
        const data = await res.json();

        if (!data) return;

        const container = document.getElementById("news-container");
        const spinner = container.querySelector(".loading-scanner");
        if (data.length > 0 && spinner) {
            spinner.remove();
        }

        renderNews(data);
        updateStats(data);
        updateHeatmap(data);
        updateAlphaPicks(data);
    } catch (e) {
        console.error("Connection Poll Failed", e);
    }
}

async function fetchMarket() {
    try {
        const res = await fetch(MARKET_URL);
        if (!res.ok) return;
        const data = await res.json();

        const moodText = document.getElementById("mood-text");
        const moodDot = document.getElementById("mood-dot");

        if (moodText) moodText.innerText = data.Mood || "NEUTRAL";

        if (moodDot) {
            const mood = data.Mood || "NEUTRAL";
            if (mood === "BULLISH") moodDot.style.backgroundColor = "var(--neon-green)";
            else if (mood === "BEARISH") moodDot.style.backgroundColor = "var(--neon-red)";
            else moodDot.style.backgroundColor = "var(--text-secondary)";
        }
    } catch (e) { }
}

function renderNews(items) {
    const container = document.getElementById("news-container");
    if (!container) return;

    if (items.length > 0 && items[0].ID !== lastSeenId) {
        lastSeenId = items[0].ID;
        if (audioEnabled && (items[0].TradingSignal.includes("STRONG") || items[0].Impact > 0.6)) {
            playSound("alert");
        }
    }

    // Process all items to handle both NEW items and UPDATES (AI)
    const reversedItems = [...items].reverse();

    reversedItems.forEach(item => {
        const existingCard = document.getElementById(`news-card-${item.ID}`);

        // 1. UPDATE EXISTING CARD (If AI arrived)
        if (existingCard) {
            // Check if AI was missing but now exists
            const hasAI = existingCard.querySelector(".ai-box");
            if (!hasAI && item.AIAnalysis) {
                console.log("⚡ AI Update for:", item.Title);
                // Update logic: preserve the card but inject AI
                // Or easier: replace innerHTML
                existingCard.innerHTML = getCardHTML(item);

                // Add flash effect for update
                existingCard.classList.add("updated-flash");
                setTimeout(() => existingCard.classList.remove("updated-flash"), 1000);
            }
            return;
        }

        // 2. CREATE NEW CARD
        const div = document.createElement("div");
        div.id = `news-card-${item.ID}`; // Unique ID for tracking

        // Class logic
        const signal = item.TradingSignal || "";
        let typeClass = "";
        if (signal.includes("BUY")) typeClass = "buy";
        else if (signal.includes("SELL")) typeClass = "sell";
        else if (signal.includes("CAUTION")) typeClass = "caution";

        div.className = `news-card ${typeClass}`;
        div.innerHTML = getCardHTML(item);

        if (container.firstChild) {
            container.insertBefore(div, container.firstChild);
        } else {
            container.appendChild(div);
        }

        renderedIds.add(item.ID);
    });
}

// Helper to generate inner content string
function getCardHTML(item) {
    let assetDisplay = item.Asset || "GEN";
    if (item.CoinSymbol && item.CoinSymbol !== "GENERAL" && item.CoinSymbol !== "MARKET") {
        assetDisplay = item.CoinSymbol;
    }

    let aiHtml = "";
    if (item.AIAnalysis) {
        aiHtml = `
        <div class="ai-badge">🤖 AI Analysis</div>
        <div class="ai-box">
           <p>${item.AIAnalysis}</p>
           <p class="advice-text">💡 ${item.AIAdvice || ""}</p>
        </div>
        `;
    }

    let timeStr = "Now";
    try {
        timeStr = new Date(item.Timestamp).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
    } catch (e) { }

    // Safe Content Creation to prevent XSS
    const metaDiv = document.createElement("div");
    metaDiv.className = "card-meta";

    const timeSpan = document.createElement("span");
    timeSpan.innerText = `${timeStr} • ${item.Source}`;

    const assetSpan = document.createElement("span");
    assetSpan.style.fontWeight = "bold";
    assetSpan.style.color = "var(--neon-blue)";
    assetSpan.innerText = assetDisplay;

    metaDiv.appendChild(timeSpan);
    metaDiv.appendChild(assetSpan);

    const titleDiv = document.createElement("div");
    titleDiv.className = "card-title";
    titleDiv.innerText = item.Title;

    // AI HTML is trusted (generated by our backend), but let's be safe.
    // For now, keep AI HTML as-is since it contains markup we might want (like bolding).
    // But Title/Source are external.

    // Combine
    const wrapper = document.createElement("div");
    wrapper.appendChild(metaDiv);
    wrapper.appendChild(titleDiv);

    if (aiHtml) {
        const aiDiv = document.createElement("div");
        aiDiv.innerHTML = aiHtml; // We trust our own AI output structure
        wrapper.appendChild(aiDiv);
    }

    return wrapper.innerHTML;
}

function updateHeatmap(items) {
    const map = {};
    items.forEach(item => {
        let sym = item.CoinSymbol;
        if (!sym || sym === "GENERAL" || sym === "MARKET" || sym === "ALT") {
            if (item.Asset !== "ALT" && item.Asset !== "ALL") sym = item.Asset;
            else return;
        }
        if (!sym) return;

        if (!map[sym]) map[sym] = 0;

        const signal = item.TradingSignal || "";
        if (signal.includes("BUY")) map[sym]++;
        if (signal.includes("SELL")) map[sym]--;
    });

    const container = document.getElementById("heatmap-container");
    if (!container) return;

    container.innerHTML = "";

    Object.keys(map).forEach(sym => {
        const score = map[sym];
        const div = document.createElement("div");
        div.innerText = sym;
        div.className = "heatmap-item";

        if (score > 0) div.classList.add("green");
        if (score < 0) div.classList.add("red");

        container.appendChild(div);
    });
}

function updateStats(items) {
    let buys = 0;
    let sells = 0;
    items.forEach(i => {
        const signal = i.TradingSignal || "";
        // COUNT ALL SIGNALS
        if (signal.includes("BUY")) buys++;
        if (signal.includes("SELL")) sells++;
    });

    const buyEl = document.getElementById("stat-buy");
    const sellEl = document.getElementById("stat-sell");

    if (buyEl) buyEl.innerText = buys;
    if (sellEl) sellEl.innerText = sells;
}


function updateAlphaPicks(items) {
    const container = document.getElementById("alpha-container");
    if (!container) return;

    // Filter for High Confidence Signals
    const alphaItems = items.filter(i =>
        i.TradingSignal === "STRONG_BUY" ||
        i.TradingSignal === "STRONG_SELL"
    );

    if (alphaItems.length === 0) {
        // Keep placeholder if empty
        if (!container.querySelector(".placeholder-text")) {
            container.innerHTML = '<p class="placeholder-text">Waiting for high confidence signals...</p>';
        }
        return;
    }

    container.innerHTML = "";
    alphaItems.forEach(item => {
        const div = document.createElement("div");
        div.className = "alpha-item glass-panel";
        div.style.marginBottom = "1rem";
        div.style.padding = "0.8rem";
        div.style.borderLeft = item.TradingSignal.includes("BUY") ? "4px solid var(--neon-green)" : "4px solid var(--neon-red)";

        div.innerHTML = `
            <div style="font-weight:bold; font-size:1.1rem; color:white;">${item.CoinSymbol || item.Asset}</div>
            <div style="font-size:0.9rem; margin-bottom:0.5rem; opacity:0.8">${item.TradingSignal.replace("_", " ")}</div>
            <div style="font-size:0.8rem; color:var(--text-secondary)">${item.AIAdvice}</div>
        `;
        container.appendChild(div);
    });
}

function playSound(type) {
    const audio = document.getElementById("alert-sound");
    if (audio) {
        audio.currentTime = 0;
        audio.play().catch(e => console.log("Audio Play Blocked"));
    }
}
