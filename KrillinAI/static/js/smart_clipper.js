// Smart Clipper Logic

let currentClips = [];
let currentToken = "";

function initSmartClipper() {
    console.log("Smart Clipper Initialized");
    const btnAnalyze = document.getElementById('btn-sc-analyze');
    if (btnAnalyze) {
        btnAnalyze.addEventListener('click', handleAnalyzeVideo);
    }

    const btnSubmit = document.getElementById('btn-sc-submit');
    if (btnSubmit) {
        btnSubmit.addEventListener('click', handleSubmitClips);
    }
}

async function handleAnalyzeVideo() {
    const urlInput = document.getElementById('sc-url-input');
    const url = urlInput.value.trim();
    if (!url) {
        alert("Please enter a video URL");
        return;
    }

    setAnalyzerState('loading');

    try {
        const res = await fetch('/api/smart_clipper/analyze', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url: url })
        });
        const json = await res.json();

        if (json.error === 0) {
            currentToken = json.data.token;
            currentClips = json.data.clips;
            renderClips(currentClips);
            setAnalyzerState('result');

            // Set video title
            document.getElementById('sc-video-title').textContent = json.data.video_title || "Unknown Video";
        } else {
            alert("Analysis failed: " + json.msg);
            setAnalyzerState('input');
        }
    } catch (e) {
        console.error(e);
        alert("Analysis error: " + e);
        setAnalyzerState('input');
    }
}

function renderClips(clips) {
    const list = document.getElementById('sc-clip-list');
    list.innerHTML = "";

    if (!clips || clips.length === 0) {
        list.innerHTML = "<div class='text-center text-gray-400'>No clips found.</div>";
        return;
    }

    clips.forEach(clip => {
        const card = document.createElement('div');
        card.className = "clip-card";
        card.innerHTML = `
            <div class="clip-header">
                <input type="checkbox" class="clip-checkbox" data-id="${clip.id}" id="clip-check-${clip.id}">
                <label for="clip-check-${clip.id}" class="clip-title">${clip.start} - ${clip.end} | ${clip.title}</label>
                <span class="clip-duration">${clip.duration}s</span>
            </div>
            <div class="clip-summary">${clip.summary}</div>
            <div class="clip-reason">${clip.reason || ''}</div>
        `;
        list.appendChild(card);
    });
}

async function handleSubmitClips() {
    if (!currentToken) {
        alert("Session expired. Please re-analyze.");
        return;
    }

    const checkboxes = document.querySelectorAll('.clip-checkbox:checked');
    const selectedIds = Array.from(checkboxes).map(cb => parseInt(cb.dataset.id));

    if (selectedIds.length === 0) {
        alert("Please select at least one clip.");
        return;
    }

    // Reuse getParams() from index.html (global scope)
    let taskParams = {};
    if (typeof getParams === 'function') {
        taskParams = getParams(); // Need to ensure we're not conflicting with main page form
        // Actually, we might want a simplified param set or reuse the main form.
        // For now, let's assume the user configures the main "Workbench" settings 
        // and we pull those settings.
    } else {
        alert("Error: Core logic not loaded.");
        return;
    }

    const btn = document.getElementById('btn-sc-submit');
    const origText = btn.textContent;
    btn.textContent = "Submitting...";
    btn.disabled = true;

    try {
        const res = await fetch('/api/smart_clipper/submit', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                token: currentToken,
                selected_clip_ids: selectedIds,
                task_params: taskParams
            })
        });
        const json = await res.json();

        if (json.error === 0) {
            alert(`Success! Created ${json.data.task_ids.length} tasks.`);
            // Redirect to history
            if (typeof switchTab === 'function') {
                switchTab('history');
            }
        } else {
            alert("Submission failed: " + json.msg);
        }
    } catch (e) {
        alert("Submission error: " + e);
    } finally {
        btn.textContent = origText;
        btn.disabled = false;
    }
}

function setAnalyzerState(state) {
    const inputSection = document.getElementById('sc-input-section');
    const loadingSection = document.getElementById('sc-loading-section');
    const resultSection = document.getElementById('sc-result-section');

    if (state === 'input') {
        inputSection.classList.remove('hidden');
        loadingSection.classList.add('hidden');
        resultSection.classList.add('hidden');
    } else if (state === 'loading') {
        inputSection.classList.add('hidden');
        loadingSection.classList.remove('hidden');
        resultSection.classList.add('hidden');
    } else if (state === 'result') {
        inputSection.classList.add('hidden');
        loadingSection.classList.add('hidden');
        resultSection.classList.remove('hidden');
    }
}

// Auto-init if DOM ready, otherwise wait
if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initSmartClipper);
} else {
    initSmartClipper();
}
