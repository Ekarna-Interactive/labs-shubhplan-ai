/**
 * Shubh Plan Web — Chat In-Line Component Widgets Library (web/widgets.js)
 * Modular UI components for embedded chat actions (Add Guest, Generate Invitation, Schedule Session).
 */

window.ChatWidgets = (function () {
  'use strict';

  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // ---------------------------------------------------------------------------
  // 1. Add Guest Component Widget
  // ---------------------------------------------------------------------------
  function renderAddGuest(container, callbacks) {
    if (!container) return;
    const widgetId = 'widget-guest-' + Math.random().toString(36).substr(2, 9);

    container.innerHTML = `
      <div class="chat-widget-card glass-card" id="${widgetId}">
        <div class="chat-widget-header">
          <span class="chat-widget-icon">👥</span>
          <div>
            <h4 class="chat-widget-title">Quick Add Guest to Roster</h4>
            <p class="chat-widget-sub">Select options below and enter guest name</p>
          </div>
        </div>

        <form class="chat-widget-form">
          <div class="chat-widget-field">
            <label class="chat-widget-label">Guest / Family Name *</label>
            <input type="text" class="form-control form-control-sm widget-guest-name" placeholder="E.g., Ramesh & Meera Iyer" required>
          </div>

          <div class="chat-widget-field">
            <label class="chat-widget-label">Category</label>
            <div class="chat-widget-pills widget-category-pills">
              <button type="button" class="chat-pill-opt selected" data-value="Family">👨‍👩‍👧 Family</button>
              <button type="button" class="chat-pill-opt" data-value="Friends">🤝 Friends</button>
              <button type="button" class="chat-pill-opt" data-value="VIPs">⭐ VIPs</button>
              <button type="button" class="chat-pill-opt" data-value="Colleagues">💼 Colleagues</button>
            </div>
          </div>

          <div class="chat-widget-row" style="display:flex; gap:12px;">
            <div class="chat-widget-field" style="flex:1;">
              <label class="chat-widget-label">RSVP Status</label>
              <div class="chat-widget-pills widget-rsvp-pills">
                <button type="button" class="chat-pill-opt selected" data-value="Confirmed">✅ Confirmed</button>
                <button type="button" class="chat-pill-opt" data-value="Pending">⏳ Pending</button>
              </div>
            </div>

            <div class="chat-widget-field" style="flex:1;">
              <label class="chat-widget-label">Plus-Ones</label>
              <div class="chat-widget-pills widget-plus-pills">
                <button type="button" class="chat-pill-opt" data-value="0">+0</button>
                <button type="button" class="chat-pill-opt selected" data-value="1">+1</button>
                <button type="button" class="chat-pill-opt" data-value="2">+2</button>
                <button type="button" class="chat-pill-opt" data-value="3">+3</button>
              </div>
            </div>
          </div>

          <div class="chat-widget-field">
            <label class="chat-widget-label">Dietary Preference</label>
            <select class="form-control form-control-sm widget-guest-notes" style="background: rgba(15, 23, 42, 0.9); color: #F8FAFC; border: 1px solid rgba(255, 255, 255, 0.15); border-radius: 8px; padding: 6px 10px; font-size: 0.85rem; width: 100%;">
              <option value="No Preference">No Specific Preference (Standard)</option>
              <option value="Vegetarian">🥗 Vegetarian (Pure Veg)</option>
              <option value="Jain">🪷 Jain (No Onion / No Garlic)</option>
              <option value="Vegan">🌱 Vegan (Plant-Based)</option>
              <option value="Halal">🌙 Halal</option>
              <option value="Eggetarian">🥚 Eggetarian</option>
              <option value="Gluten-Free">🌾 Gluten-Free / Special Allergies</option>
            </select>
          </div>

          <button type="submit" class="btn btn-sm btn-accent btn-block widget-submit-btn" style="margin-top:8px;">
            <span>➕ Add Guest to Roster</span>
          </button>
        </form>

        <div class="widget-status-msg hidden" style="margin-top:8px; font-size:0.82rem; font-weight:600;"></div>
      </div>
    `;

    const widget = document.getElementById(widgetId);
    if (!widget) return;

    // Attach pill selection listeners
    widget.querySelectorAll('.widget-category-pills .chat-pill-opt').forEach(btn => {
      btn.addEventListener('click', () => {
        widget.querySelectorAll('.widget-category-pills .chat-pill-opt').forEach(b => b.classList.remove('selected'));
        btn.classList.add('selected');
      });
    });

    widget.querySelectorAll('.widget-rsvp-pills .chat-pill-opt').forEach(btn => {
      btn.addEventListener('click', () => {
        widget.querySelectorAll('.widget-rsvp-pills .chat-pill-opt').forEach(b => b.classList.remove('selected'));
        btn.classList.add('selected');
      });
    });

    widget.querySelectorAll('.widget-plus-pills .chat-pill-opt').forEach(btn => {
      btn.addEventListener('click', () => {
        widget.querySelectorAll('.widget-plus-pills .chat-pill-opt').forEach(b => b.classList.remove('selected'));
        btn.classList.add('selected');
      });
    });

    // Handle form submit
    const form = widget.querySelector('.chat-widget-form');
    const submitBtn = widget.querySelector('.widget-submit-btn');
    const statusMsg = widget.querySelector('.widget-status-msg');

    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const name = widget.querySelector('.widget-guest-name').value.trim();
      if (!name) return;

      const category = widget.querySelector('.widget-category-pills .selected')?.getAttribute('data-value') || 'Family';
      const rsvpStatus = widget.querySelector('.widget-rsvp-pills .selected')?.getAttribute('data-value') || 'Confirmed';
      const plusOnes = parseInt(widget.querySelector('.widget-plus-pills .selected')?.getAttribute('data-value') || '0', 10);
      const dietaryRequirements = widget.querySelector('.widget-guest-notes').value.trim() || 'No Restrictions';

      submitBtn.disabled = true;
      submitBtn.innerHTML = '<span>⏳ Saving Guest...</span>';

      try {
        const res = await fetch('/api/guests', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name,
            category,
            rsvpStatus,
            plusOnes,
            dietaryRequirements,
            notes: 'Added via In-Chat Component Widget'
          })
        });

        if (res.ok) {
          statusMsg.className = 'widget-status-msg text-success';
          statusMsg.innerHTML = `✅ Successfully added <strong>${escapeHtml(name)}</strong> (${category}) to guest roster!`;
          statusMsg.classList.remove('hidden');
          form.style.display = 'none';

          if (callbacks && typeof callbacks.onSuccess === 'function') {
            callbacks.onSuccess('guest');
          }
        } else {
          throw new Error('Failed to add guest');
        }
      } catch (err) {
        statusMsg.className = 'widget-status-msg text-danger';
        statusMsg.textContent = '⚠️ Error adding guest. Please try again.';
        statusMsg.classList.remove('hidden');
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<span>➕ Add Guest to Roster</span>';
      }
    });
  }

  // ---------------------------------------------------------------------------
  // 2. Generate Invitation Component Widget
  // ---------------------------------------------------------------------------
  function renderGenerateInvitation(container, callbacks) {
    if (!container) return;
    const widgetId = 'widget-invitation-' + Math.random().toString(36).substr(2, 9);

    container.innerHTML = `
      <div class="chat-widget-card glass-card" id="${widgetId}">
        <div class="chat-widget-header">
          <span class="chat-widget-icon">🎨</span>
          <div>
            <h4 class="chat-widget-title">Quick Generate Invitation Card</h4>
            <p class="chat-widget-sub">Select style preset & aspect ratio for AI artwork</p>
          </div>
        </div>

        <form class="chat-widget-form">
          <div class="chat-widget-field">
            <label class="chat-widget-label">Aesthetic Style Preset</label>
            <div class="chat-widget-pills widget-style-pills">
              <button type="button" class="chat-pill-opt selected" data-value="Clay 3D">🏺 Clay 3D</button>
              <button type="button" class="chat-pill-opt" data-value="South Indian">🛕 South Indian</button>
              <button type="button" class="chat-pill-opt" data-value="Mughal">🕌 Mughal</button>
              <button type="button" class="chat-pill-opt" data-value="Paper Cut">✂️ Paper Cut</button>
              <button type="button" class="chat-pill-opt" data-value="Pop Art">💥 Pop Art</button>
              <button type="button" class="chat-pill-opt" data-value="Minimalist Gold">✨ Minimalist Gold</button>
              <button type="button" class="chat-pill-opt" data-value="Watercolor">🎨 Watercolor</button>
            </div>
          </div>

          <div class="chat-widget-field">
            <label class="chat-widget-label">Aspect Ratio</label>
            <div class="chat-widget-pills widget-aspect-pills">
              <button type="button" class="chat-pill-opt selected" data-value="4:5">📱 4:5 Feed</button>
              <button type="button" class="chat-pill-opt" data-value="9:16">📲 9:16 Story</button>
              <button type="button" class="chat-pill-opt" data-value="1:1">🟦 1:1 Post</button>
              <button type="button" class="chat-pill-opt" data-value="16:9">🖼️ 16:9 Banner</button>
            </div>
          </div>

          <div class="chat-widget-field">
            <label class="chat-widget-label">Custom Visual Notes / Motifs (Optional)</label>
            <input type="text" class="form-control form-control-sm widget-invite-prompt" placeholder="E.g., Baby lion with marigold garlands...">
          </div>

          <button type="submit" class="btn btn-sm btn-accent btn-block widget-submit-btn" style="margin-top:8px;">
            <span>✨ Synthesize AI Prompt Suggestions</span>
          </button>
        </form>

        <div class="widget-prompts-container hidden" style="margin-top:12px;"></div>
        <div class="widget-status-msg hidden" style="margin-top:8px; font-size:0.82rem; font-weight:600;"></div>
      </div>
    `;

    const widget = document.getElementById(widgetId);
    if (!widget) return;

    // Attach pill selection listeners
    widget.querySelectorAll('.widget-style-pills .chat-pill-opt').forEach(btn => {
      btn.addEventListener('click', () => {
        widget.querySelectorAll('.widget-style-pills .chat-pill-opt').forEach(b => b.classList.remove('selected'));
        btn.classList.add('selected');
      });
    });

    widget.querySelectorAll('.widget-aspect-pills .chat-pill-opt').forEach(btn => {
      btn.addEventListener('click', () => {
        widget.querySelectorAll('.widget-aspect-pills .chat-pill-opt').forEach(b => b.classList.remove('selected'));
        btn.classList.add('selected');
      });
    });

    // Handle form submit (Step 1: Synthesize Prompt Suggestions)
    const form = widget.querySelector('.chat-widget-form');
    const submitBtn = widget.querySelector('.widget-submit-btn');
    const promptsContainer = widget.querySelector('.widget-prompts-container');
    const statusMsg = widget.querySelector('.widget-status-msg');

    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const styleTheme = widget.querySelector('.widget-style-pills .selected')?.getAttribute('data-value') || 'Clay 3D';
      const aspectRatio = widget.querySelector('.widget-aspect-pills .selected')?.getAttribute('data-value') || '4:5';
      const customPrompt = widget.querySelector('.widget-invite-prompt').value.trim();

      submitBtn.disabled = true;
      submitBtn.innerHTML = '<span>⏳ Synthesizing 4 AI Prompt Suggestions...</span>';
      statusMsg.classList.add('hidden');

      try {
        const res = await fetch('/api/flows/invitationPromptSuggestionsFlow', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            stylePreset: styleTheme,
            aspectRatio: aspectRatio,
            customElements: customPrompt ? [customPrompt] : [],
            primaryColor: '#D4AF37',
            typography: 'Cinzel Decorative & Outfit'
          })
        });

        if (res.ok) {
          const data = await res.json();
          const suggestions = data.suggestions || [];

          if (suggestions.length === 0) {
            throw new Error('No prompt suggestions generated');
          }

          form.style.display = 'none';

          promptsContainer.innerHTML = `
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:12px;">
              <h5 style="margin:0; font-size:0.88rem; color:#D4AF37; font-weight:700;">✨ 4 Gemini AI Prompt Suggestions (${escapeHtml(styleTheme)} • ${aspectRatio}):</h5>
              <button type="button" class="btn btn-sm btn-outline widget-back-btn" style="font-size:0.75rem; padding:2px 8px; border-radius:12px;">← Edit Options</button>
            </div>

            <div style="display:flex; flex-direction:column; gap:10px;">
              ${suggestions.map((sug, idx) => {
                const cleanTitle = (sug.title || `Concept Option ${idx + 1}`).replace(/^Option \d+:\s*/i, '');
                const promptText = sug.promptText || sug.prompt || sug.description || 'Bespoke invitation card artwork prompt';
                return `
                  <div class="widget-sug-card" style="background:rgba(255,255,255,0.03); border:1px solid rgba(255,255,255,0.1); border-radius:10px; padding:10px 12px;">
                    <div style="font-weight:700; color:#F3E5AB; font-size:0.83rem; margin-bottom:4px;">Option ${idx + 1}: ${escapeHtml(cleanTitle)}</div>
                    <p style="font-size:0.78rem; color:#CBD5E1; margin-bottom:8px; line-height:1.4;">${escapeHtml(promptText)}</p>
                    <button type="button" class="btn btn-sm btn-accent widget-gen-art-btn" data-prompt="${escapeHtml(promptText)}" data-style="${escapeHtml(styleTheme)}" data-aspect="${escapeHtml(aspectRatio)}" style="font-size:0.78rem; padding:4px 14px; border-radius:16px;">
                      🎨 Generate Option ${idx + 1} Artwork
                    </button>
                  </div>
                `;
              }).join('')}
            </div>
          `;
          promptsContainer.classList.remove('hidden');

          // Back Button listener
          promptsContainer.querySelector('.widget-back-btn').addEventListener('click', () => {
            promptsContainer.classList.add('hidden');
            form.style.display = 'block';
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<span>✨ Synthesize AI Prompt Suggestions</span>';
          });

          // Generate Option Artwork listeners
          promptsContainer.querySelectorAll('.widget-gen-art-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
              const chosenPrompt = btn.getAttribute('data-prompt');
              const chosenStyle = btn.getAttribute('data-style');
              const chosenAspect = btn.getAttribute('data-aspect');

              // Disable all sibling buttons and apply animated loading state
              promptsContainer.querySelectorAll('.widget-gen-art-btn').forEach(b => {
                b.disabled = true;
              });
              btn.classList.add('btn-rendering-active');
              btn.innerHTML = '<span class="typing-indicator" style="display:inline-flex; gap:3px; margin-right:6px;"><span></span><span></span><span></span></span> <span>⏳ Synthesizing & Rendering PNG...</span>';

              try {
                const genRes = await fetch('/api/flows/invitationGeneratorFlow', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    aestheticTheme: chosenStyle,
                    styleTheme: chosenStyle,
                    aspectRatio: chosenAspect,
                    promptText: chosenPrompt,
                    prompt: chosenPrompt,
                    customElements: customPrompt ? [customPrompt] : [],
                    primaryColor: '#D4AF37',
                    typography: 'Cinzel Decorative & Outfit',
                    specialInstructions: ''
                  })
                });

                if (genRes.ok) {
                  const genData = await genRes.json();
                  const imgUrl = genData.mainConcept?.imageUrl || '/assets/card_1787724499766185100.png';

                  statusMsg.className = 'widget-status-msg text-success';
                  statusMsg.innerHTML = `
                    <div style="text-align:center; padding:8px;">
                      <p style="margin-bottom:8px;">🎉 Generated <strong>${escapeHtml(chosenStyle)}</strong> (${chosenAspect}) Invitation Card!</p>
                      <div class="widget-img-preview-box" style="border-radius:10px; overflow:hidden; border:1px solid rgba(212,175,55,0.4); max-height:300px; cursor:pointer;" title="Click for full-screen preview">
                        <img src="${imgUrl}" alt="Generated Card" style="width:100%; height:100%; object-fit:contain; display:block;">
                      </div>
                    </div>
                  `;
                  statusMsg.classList.remove('hidden');
                  promptsContainer.classList.add('hidden');

                  const previewBox = statusMsg.querySelector('.widget-img-preview-box');
                  if (previewBox) {
                    previewBox.addEventListener('click', () => {
                      if (window.openCardPreview) {
                        window.openCardPreview(imgUrl, 'Generated Card Artwork', chosenStyle);
                      }
                    });
                  }

                  // Auto-scroll chat view to center the generated card image
                  setTimeout(() => {
                    if (previewBox) {
                      previewBox.scrollIntoView({ behavior: 'smooth', block: 'center' });
                    } else if (statusMsg) {
                      statusMsg.scrollIntoView({ behavior: 'smooth', block: 'end' });
                    }
                  }, 80);

                  if (callbacks && typeof callbacks.onSuccess === 'function') {
                    callbacks.onSuccess('design');
                  }
                } else {
                  throw new Error('Failed to generate artwork');
                }
              } catch (err) {
                statusMsg.className = 'widget-status-msg text-danger';
                statusMsg.textContent = `⚠️ Error generating artwork. Please try again.`;
                statusMsg.classList.remove('hidden');
                promptsContainer.querySelectorAll('.widget-gen-art-btn').forEach(b => {
                  b.disabled = false;
                  b.classList.remove('btn-rendering-active');
                  b.innerHTML = '🎨 Generate Artwork';
                });
              }
            });
          });

        } else {
          throw new Error('Failed to synthesize prompt suggestions');
        }
      } catch (err) {
        statusMsg.className = 'widget-status-msg text-danger';
        statusMsg.textContent = `⚠️ Error synthesizing prompt suggestions. Please try again.`;
        statusMsg.classList.remove('hidden');
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<span>✨ Synthesize AI Prompt Suggestions</span>';
      }
    });
  }

  // ---------------------------------------------------------------------------
  // 3. Schedule Session Component Widget
  // ---------------------------------------------------------------------------
  function renderScheduleSession(container, callbacks) {
    if (!container) return;
    const widgetId = 'widget-itinerary-' + Math.random().toString(36).substr(2, 9);

    container.innerHTML = `
      <div class="chat-widget-card glass-card" id="${widgetId}">
        <div class="chat-widget-header">
          <span class="chat-widget-icon">📅</span>
          <div>
            <h4 class="chat-widget-title">Quick Add Session to Itinerary</h4>
            <p class="chat-widget-sub">Enter session details to update event agenda</p>
          </div>
        </div>

        <form class="chat-widget-form">
          <div class="chat-widget-row" style="display:flex; gap:12px;">
            <div class="chat-widget-field" style="flex:1;">
              <label class="chat-widget-label">Session Title *</label>
              <input type="text" class="form-control form-control-sm widget-session-title" placeholder="E.g., Welcome Drinks & Snacks" required>
            </div>
            <div class="chat-widget-field" style="width:130px;">
              <label class="chat-widget-label">Time *</label>
              <input type="text" class="form-control form-control-sm widget-session-time" placeholder="E.g., 06:30 PM" required>
            </div>
          </div>

          <div class="chat-widget-field">
            <label class="chat-widget-label">Location / Stage</label>
            <input type="text" class="form-control form-control-sm widget-session-loc" placeholder="E.g., Entrance Foyer / Main Mandapam">
          </div>

          <div class="chat-widget-field">
            <label class="chat-widget-label">Description / Notes</label>
            <input type="text" class="form-control form-control-sm widget-session-desc" placeholder="E.g., Traditional welcome with rose water and music">
          </div>

          <button type="submit" class="btn btn-sm btn-accent btn-block widget-submit-btn" style="margin-top:8px;">
            <span>📅 Add Session to Itinerary</span>
          </button>
        </form>

        <div class="widget-status-msg hidden" style="margin-top:8px; font-size:0.82rem; font-weight:600;"></div>
      </div>
    `;

    const widget = document.getElementById(widgetId);
    if (!widget) return;

    const form = widget.querySelector('.chat-widget-form');
    const submitBtn = widget.querySelector('.widget-submit-btn');
    const statusMsg = widget.querySelector('.widget-status-msg');

    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const title = widget.querySelector('.widget-session-title').value.trim();
      const time = widget.querySelector('.widget-session-time').value.trim();
      const location = widget.querySelector('.widget-session-loc').value.trim();
      const description = widget.querySelector('.widget-session-desc').value.trim();

      if (!title || !time) return;

      submitBtn.disabled = true;
      submitBtn.innerHTML = '<span>⏳ Saving Session...</span>';

      try {
        const res = await fetch('/api/itinerary', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title, time, location, description })
        });

        if (res.ok) {
          statusMsg.className = 'widget-status-msg text-success';
          statusMsg.innerHTML = `✅ Successfully scheduled <strong>${escapeHtml(title)}</strong> at <strong>${escapeHtml(time)}</strong>!`;
          statusMsg.classList.remove('hidden');
          form.style.display = 'none';

          if (callbacks && typeof callbacks.onSuccess === 'function') {
            callbacks.onSuccess('itinerary');
          }
        } else {
          throw new Error('Failed to schedule session');
        }
      } catch (err) {
        statusMsg.className = 'widget-status-msg text-danger';
        statusMsg.textContent = '⚠️ Error scheduling session. Please try again.';
        statusMsg.classList.remove('hidden');
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<span>📅 Add Session to Itinerary</span>';
      }
    });
  }

  // ---------------------------------------------------------------------------
  // 4. Venue Confirmation Component Widget
  // ---------------------------------------------------------------------------
  function renderSelectVenue(container, callbacks) {
    if (!container) return;
    const query = container.getAttribute('data-query') || container.getAttribute('data-venue-query') || '';
    const widgetId = 'widget-venue-' + Math.random().toString(36).substr(2, 9);

    container.innerHTML = `
      <div class="chat-widget-card glass-card" id="${widgetId}">
        <div class="chat-widget-header">
          <span class="chat-widget-icon">📍</span>
          <div>
            <h4 class="chat-widget-title">Confirm Venue Location</h4>
            <p class="chat-widget-sub">Review matching Google Places options and confirm venue selection</p>
          </div>
        </div>

        <div class="widget-venue-results" style="margin-top: 10px;">
          <div class="venue-search-loading" style="font-size: 0.85rem; color: #94a3b8; padding: 12px; text-align: center;">
            <span class="typing-indicator" style="display:inline-flex; gap:3px; margin-right:6px;"><span></span><span></span><span></span></span>
            <span>Searching Google Places for "${escapeHtml(query)}"...</span>
          </div>
        </div>

        <div class="widget-status-msg hidden" style="margin-top:10px; font-size:0.85rem; font-weight:600; text-align:center;"></div>
      </div>
    `;

    const widget = document.getElementById(widgetId);
    if (!widget) return;

    const resultsBox = widget.querySelector('.widget-venue-results');
    const statusMsg = widget.querySelector('.widget-status-msg');

    fetch(`/api/venue/search?query=${encodeURIComponent(query)}`)
      .then(res => res.json())
      .then(data => {
        if (!data || !data.results || data.results.length === 0) {
          resultsBox.innerHTML = `<p style="font-size: 0.85rem; color: #f87171; text-align:center; padding:8px;">⚠️ No Google Places location found matching "${escapeHtml(query)}".</p>`;
          return;
        }

        resultsBox.innerHTML = data.results.map((vd, idx) => `
          <div class="venue-option-card" style="background: rgba(255, 255, 255, 0.04); border: 1px solid rgba(255, 255, 255, 0.1); border-radius: 12px; padding: 12px; margin-bottom: 10px;">
            ${vd.venue_photo_url ? `
              <div style="height: 120px; overflow: hidden; border-radius: 8px; margin-bottom: 10px;">
                <img src="${escapeHtml(vd.venue_photo_url)}" alt="${escapeHtml(vd.primary_venue)}" style="width: 100%; height: 100%; object-fit: cover;">
              </div>
            ` : ''}
            <h5 style="font-size: 0.95rem; font-weight: 700; color: #f8fafc; margin-bottom: 4px;">${escapeHtml(vd.primary_venue || query)}</h5>
            <p style="font-size: 0.82rem; color: #94a3b8; margin-bottom: 8px; line-height: 1.4;">📍 ${escapeHtml(vd.venue_formatted_address || vd.address)}</p>
            <div style="display: flex; gap: 8px; align-items: center; justify-content: space-between;">
              ${vd.google_map_url ? `<a href="${escapeHtml(vd.google_map_url)}" target="_blank" class="btn btn-sm btn-outline" style="font-size:0.75rem; padding: 4px 8px;">🗺️ View Map</a>` : '<span></span>'}
              <button type="button" class="btn btn-sm btn-accent widget-confirm-venue-btn" data-venue="${escapeHtml(vd.primary_venue)}" data-location="${escapeHtml(vd.address)}" data-details='${escapeHtml(JSON.stringify(vd))}'>
                ✅ Confirm & Set Venue
              </button>
            </div>
          </div>
        `).join('');

        resultsBox.querySelectorAll('.widget-confirm-venue-btn').forEach(btn => {
          btn.addEventListener('click', async () => {
            const venueName = btn.getAttribute('data-venue');
            const locName = btn.getAttribute('data-location');
            let venueDetailsObj = {};
            try {
              venueDetailsObj = JSON.parse(btn.getAttribute('data-details'));
            } catch (e) {}

            btn.disabled = true;
            btn.innerHTML = '⏳ Setting Venue...';

            try {
              const postRes = await fetch('/api/event', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  venue: venueName,
                  location: locName,
                  venueDetails: venueDetailsObj
                })
              });

              if (postRes.ok) {
                resultsBox.style.display = 'none';
                statusMsg.className = 'widget-status-msg text-success';
                statusMsg.innerHTML = `🎉 Confirmed! Event venue set to <strong>${escapeHtml(venueName)}</strong> (<em>${escapeHtml(locName)}</em>).`;
                statusMsg.classList.remove('hidden');

                if (callbacks && typeof callbacks.onSuccess === 'function') {
                  callbacks.onSuccess('event');
                }
              } else {
                throw new Error('Failed to update venue');
              }
            } catch (err) {
              btn.disabled = false;
              btn.innerHTML = '✅ Confirm & Set Venue';
              statusMsg.className = 'widget-status-msg text-danger';
              statusMsg.textContent = '⚠️ Error confirming venue. Please try again.';
              statusMsg.classList.remove('hidden');
            }
          });
        });
      })
      .catch(err => {
        resultsBox.innerHTML = `<p style="font-size: 0.85rem; color: #f87171; text-align:center; padding:8px;">⚠️ Failed to load Google Places options.</p>`;
      });
  }

  // ---------------------------------------------------------------------------
  // 5. Component Mount Helper
  // ---------------------------------------------------------------------------
  function mountAll(parentContainer, callbacks) {
    if (!parentContainer) return;

    const guestMounts = parentContainer.querySelectorAll('.widget-mount-add-guest');
    guestMounts.forEach(el => {
      if (!el.hasAttribute('data-mounted')) {
        el.setAttribute('data-mounted', 'true');
        renderAddGuest(el, callbacks);
      }
    });

    const inviteMounts = parentContainer.querySelectorAll('.widget-mount-generate-invitation');
    inviteMounts.forEach(el => {
      if (!el.hasAttribute('data-mounted')) {
        el.setAttribute('data-mounted', 'true');
        renderGenerateInvitation(el, callbacks);
      }
    });

    const itinMounts = parentContainer.querySelectorAll('.widget-mount-schedule-session');
    itinMounts.forEach(el => {
      if (!el.hasAttribute('data-mounted')) {
        el.setAttribute('data-mounted', 'true');
        renderScheduleSession(el, callbacks);
      }
    });

    const venueMounts = parentContainer.querySelectorAll('.widget-mount-select-venue');
    venueMounts.forEach(el => {
      if (!el.hasAttribute('data-mounted')) {
        el.setAttribute('data-mounted', 'true');
        renderSelectVenue(el, callbacks);
      }
    });
  }

  return {
    renderAddGuest: renderAddGuest,
    renderGenerateInvitation: renderGenerateInvitation,
    renderScheduleSession: renderScheduleSession,
    renderSelectVenue: renderSelectVenue,
    mountAll: mountAll
  };
})();
