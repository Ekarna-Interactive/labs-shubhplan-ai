// Intercept fetch to attach client-side API Key & Session headers
const originalFetch = window.fetch;
window.fetch = async function (url, options = {}) {
  options.headers = options.headers || {};
  const clientApiKey = localStorage.getItem('shubh_gemini_api_key');
  if (clientApiKey) {
    if (options.headers instanceof Headers) {
      options.headers.set('X-Gemini-API-Key', clientApiKey);
    } else {
      options.headers['X-Gemini-API-Key'] = clientApiKey;
    }
  }
  const clientMapsKey = localStorage.getItem('shubh_maps_api_key');
  if (clientMapsKey) {
    if (options.headers instanceof Headers) {
      options.headers.set('X-Google-Maps-API-Key', clientMapsKey);
    } else {
      options.headers['X-Google-Maps-API-Key'] = clientMapsKey;
    }
  }
  const clientSessionToken = localStorage.getItem('shubh_session_token');
  if (clientSessionToken) {
    if (options.headers instanceof Headers) {
      options.headers.set('X-Session-ID', clientSessionToken);
    } else {
      options.headers['X-Session-ID'] = clientSessionToken;
    }
  }
  return originalFetch.call(this, url, options);
};

document.addEventListener('DOMContentLoaded', () => {
  // Global State
  const state = {
    event: null,
    guests: [],
    itinerary: [],
    designs: [],
    sessionId: '',
    user: null,
    appMode: 'demo',
    hasServerAPIKey: false,
    activeTab: 'tab-copilot',
  };

  // Initialize Core Controllers
  initAuthAndMode();
  initApiKeyModal();
  initEventProfileModal();
  initAccountSettingsModal();
  initTabs();
  initChat();
  initStudio();
  initRoster();
  initItinerary();

  // Initial Load
  loadAllData();

  // ---------------------------------------------------------------------------
  // 1. Operating Modes & Authentication Controller
  // ---------------------------------------------------------------------------
  async function initAuthAndMode() {
    try {
      const modeRes = await fetch('/api/mode');
      if (modeRes.ok) {
        const modeData = await modeRes.json();
        state.appMode = modeData.mode || 'demo';
        state.hasServerAPIKey = modeData.hasServerAPIKey;

        const modeBadgeText = document.getElementById('app-mode-text');
        if (modeBadgeText) {
          modeBadgeText.textContent = state.appMode === 'demo' ? 'DEMO MODE' : 'SERVER MODE';
        }
        updateKeyDotStatus();
      }

      const meRes = await fetch('/api/auth/me');
      if (meRes.ok) {
        state.user = await meRes.json();
        updateUserUI();
      } else {
        updateUserUI();
      }
    } catch (e) {
      console.warn('Auth check error:', e);
    }

    const authModal = document.getElementById('auth-modal');
    const authMenuBtn = document.getElementById('auth-menu-btn');
    const closeAuthBtn = document.getElementById('close-auth-modal-btn');
    const userDropdown = document.getElementById('user-dropdown-menu');
    const menuLogoutBtn = document.getElementById('menu-logout-btn');
    const menuEventProfileBtn = document.getElementById('menu-event-profile-btn');
    const menuAccountSettingsBtn = document.getElementById('menu-account-settings-btn');
    const eventPill = document.getElementById('event-pill');
    const tabLoginBtn = document.getElementById('tab-login-btn');
    const tabSignupBtn = document.getElementById('tab-signup-btn');
    const loginForm = document.getElementById('login-form');
    const signupForm = document.getElementById('signup-form');
    const guestDemoBtn = document.getElementById('guest-demo-btn');
    const fillDemoBtn = document.getElementById('fill-demo-credentials-btn');

    if (authMenuBtn) {
      authMenuBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        if (state.user) {
          if (userDropdown) userDropdown.classList.toggle('hidden');
        } else {
          if (authModal) authModal.classList.add('open');
        }
      });
    }

    document.addEventListener('click', (e) => {
      if (userDropdown && !userDropdown.contains(e.target) && e.target !== authMenuBtn) {
        userDropdown.classList.add('hidden');
      }
    });

    if (menuLogoutBtn) {
      menuLogoutBtn.addEventListener('click', () => {
        if (userDropdown) userDropdown.classList.add('hidden');
        logoutUser();
      });
    }

    if (menuEventProfileBtn) {
      menuEventProfileBtn.addEventListener('click', () => {
        if (userDropdown) userDropdown.classList.add('hidden');
        openEventProfileModal();
      });
    }

    if (eventPill) {
      eventPill.addEventListener('click', () => {
        openEventProfileModal();
      });
    }

    if (menuAccountSettingsBtn) {
      menuAccountSettingsBtn.addEventListener('click', () => {
        if (userDropdown) userDropdown.classList.add('hidden');
        openAccountSettingsModal();
      });
    }

    if (closeAuthBtn && authModal) {
      closeAuthBtn.addEventListener('click', () => authModal.classList.remove('open'));
    }

    if (fillDemoBtn) {
      fillDemoBtn.addEventListener('click', () => {
        const emailIn = document.getElementById('login-email');
        const passIn = document.getElementById('login-password');
        if (emailIn) emailIn.value = 'admin@shubhplan.ai';
        if (passIn) passIn.value = 'shubh2026';
      });
    }

    if (tabLoginBtn && tabSignupBtn && loginForm && signupForm) {
      tabLoginBtn.addEventListener('click', () => {
        tabLoginBtn.classList.add('active');
        tabSignupBtn.classList.remove('active');
        loginForm.style.display = 'block';
        signupForm.style.display = 'none';
      });
      tabSignupBtn.addEventListener('click', () => {
        tabSignupBtn.classList.add('active');
        tabLoginBtn.classList.remove('active');
        signupForm.style.display = 'block';
        loginForm.style.display = 'none';
      });
    }

    if (loginForm) {
      loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('login-email').value;
        const password = document.getElementById('login-password').value;
        const errBox = document.getElementById('login-error');
        if (errBox) errBox.classList.add('hidden');

        try {
          const res = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
          });
          const data = await res.json();
          if (res.ok && data.session) {
            localStorage.setItem('shubh_session_token', data.session.token);
            state.user = data.session;
            updateUserUI();
            if (authModal) authModal.classList.remove('open');
            loadAllData();
          } else {
            if (errBox) {
              errBox.textContent = data.error || 'Login failed.';
              errBox.classList.remove('hidden');
            }
          }
        } catch (err) {
          if (errBox) {
            errBox.textContent = 'Network or authentication error.';
            errBox.classList.remove('hidden');
          }
        }
      });
    }

    if (signupForm) {
      signupForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const fullName = document.getElementById('signup-name').value;
        const email = document.getElementById('signup-email').value;
        const password = document.getElementById('signup-password').value;
        const errBox = document.getElementById('signup-error');
        if (errBox) errBox.classList.add('hidden');

        try {
          const res = await fetch('/api/auth/signup', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ fullName, email, password })
          });
          const data = await res.json();
          if (res.ok && data.session) {
            localStorage.setItem('shubh_session_token', data.session.token);
            state.user = data.session;
            updateUserUI();
            if (authModal) authModal.classList.remove('open');
            loadAllData();
          } else {
            if (errBox) {
              errBox.textContent = data.error || 'Account creation failed.';
              errBox.classList.remove('hidden');
            }
          }
        } catch (err) {
          if (errBox) {
            errBox.textContent = 'Registration error occurred.';
            errBox.classList.remove('hidden');
          }
        }
      });
    }

    if (guestDemoBtn) {
      guestDemoBtn.addEventListener('click', async () => {
        try {
          const res = await fetch('/api/auth/guest-demo', { method: 'POST' });
          const data = await res.json();
          if (res.ok && data.session) {
            localStorage.setItem('shubh_session_token', data.session.token);
            state.user = data.session;
            updateUserUI();
            if (authModal) authModal.classList.remove('open');
            loadAllData();
          }
        } catch (err) {
          console.error('Guest demo error:', err);
        }
      });
    }
  }

  async function logoutUser() {
    try {
      await fetch('/api/auth/logout', { method: 'POST' });
    } catch (e) {
      console.warn('Logout error:', e);
    }
    localStorage.removeItem('shubh_session_token');
    state.user = null;
    updateUserUI();
  }

  function updateUserUI() {
    const authText = document.getElementById('auth-btn-text');
    const authModal = document.getElementById('auth-modal');
    const closeAuthBtn = document.getElementById('close-auth-modal-btn');
    const dropEmail = document.getElementById('dropdown-user-email');
    const dropRole = document.getElementById('dropdown-user-role');

    if (state.user) {
      const email = state.user.userEmail || state.user.email || 'User';
      const role = state.user.userRole || state.user.role || 'Planner';
      const displayStr = email.includes('@') ? email.split('@')[0] : email;

      if (authText) authText.textContent = `👤 ${displayStr}`;
      if (dropEmail) dropEmail.textContent = email;
      if (dropRole) dropRole.textContent = `Role: ${role.toUpperCase()}`;

      if (authModal) authModal.classList.remove('open');
      if (closeAuthBtn) closeAuthBtn.classList.remove('hidden');
    } else {
      if (authText) authText.textContent = 'Sign Up / Login';
      if (authModal) authModal.classList.add('open');
      if (closeAuthBtn) closeAuthBtn.classList.add('hidden');
    }
  }

  // ---------------------------------------------------------------------------
  // 2. Data Loader
  // ---------------------------------------------------------------------------
  async function loadAllData() {
    await fetchEvent();
    await fetchGuests();
    await fetchItinerary();
    await fetchDesigns();
  }

  async function fetchEvent() {
    try {
      const res = await fetch('/api/event');
      if (res.ok) {
        state.event = await res.json();
        renderEventProfile();
      }
    } catch (e) {
      console.warn('Fetch event error:', e);
    }
  }

  function renderEventProfile() {
    if (!state.event) return;
    const e = state.event;
    const headerName = document.getElementById('header-event-name');
    if (headerName) headerName.textContent = e.title || "Shubh Plan Workspace";

    const titleEl = document.getElementById('summary-title');
    const typeEl = document.getElementById('summary-type');
    const dateEl = document.getElementById('summary-date');
    const hostsEl = document.getElementById('summary-hosts');
    const themeEl = document.getElementById('summary-theme');
    const countEl = document.getElementById('summary-count');

    if (titleEl) titleEl.textContent = e.title || '-';
    if (typeEl) typeEl.textContent = e.eventType || '-';
    if (dateEl) dateEl.textContent = e.date ? formatHumanReadableDate(e.date) : '-';
    if (hostsEl) hostsEl.textContent = e.hostNames || '-';
    if (themeEl) themeEl.textContent = e.aestheticTheme || '-';
    if (countEl) countEl.textContent = e.targetGuestCount ? `${e.targetGuestCount} guests` : '-';

    const vName = document.getElementById('venue-name-display');
    const vAddr = document.getElementById('venue-address-display');
    const vImg = document.getElementById('venue-photo-img');
    const vMaps = document.getElementById('venue-maps-link');
    const vDir = document.getElementById('venue-directions-link');

    if (vName) vName.textContent = e.venue || 'Venue Unconfirmed';
    if (vAddr) {
      vAddr.textContent = (e.venueDetails && e.venueDetails.venue_formatted_address)
        ? e.venueDetails.venue_formatted_address
        : (e.location || 'Search or tell AI Assistant your venue location');
    }
    if (vImg) {
      vImg.src = (e.venueDetails && e.venueDetails.venue_photo_url)
        ? e.venueDetails.venue_photo_url
        : 'https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=800&q=80';
    }
    if (vMaps && e.venueDetails && e.venueDetails.google_map_url) {
      vMaps.href = e.venueDetails.google_map_url;
      vMaps.style.display = 'inline-flex';
    }
    if (vDir && e.venueDetails && e.venueDetails.google_map_directions_url) {
      vDir.href = e.venueDetails.google_map_directions_url;
      vDir.style.display = 'inline-flex';
    }
  }

  // ---------------------------------------------------------------------------
  // 3. API Key Settings Modal Controller
  // ---------------------------------------------------------------------------
  function initApiKeyModal() {
    const apikeyModal = document.getElementById('apikey-modal');
    const openKeyBtn = document.getElementById('open-apikey-modal-btn');
    const closeKeyBtn = document.getElementById('close-apikey-modal-btn');
    const apikeyForm = document.getElementById('apikey-form');
    const keyInput = document.getElementById('user-gemini-key-input');
    const clearKeyBtn = document.getElementById('clear-apikey-btn');

    updateKeyDotStatus();

    if (openKeyBtn && apikeyModal) {
      openKeyBtn.addEventListener('click', () => {
        const savedGeminiKey = localStorage.getItem('shubh_gemini_api_key') || '';
        const savedMapsKey = localStorage.getItem('shubh_maps_api_key') || '';
        if (keyInput) keyInput.value = savedGeminiKey;
        const mapsInput = document.getElementById('user-maps-key-input');
        if (mapsInput) mapsInput.value = savedMapsKey;

        const msg = document.getElementById('key-status-msg');
        if (msg) {
          if (savedGeminiKey || state.hasServerAPIKey) {
            msg.textContent = '✅ API Keys active in browser memory.';
            msg.className = 'key-status-msg success';
            msg.classList.remove('hidden');
          } else {
            msg.classList.add('hidden');
          }
        }
        apikeyModal.classList.add('open');
      });
    }

    if (closeKeyBtn && apikeyModal) {
      closeKeyBtn.addEventListener('click', () => apikeyModal.classList.remove('open'));
    }

    if (clearKeyBtn) {
      clearKeyBtn.addEventListener('click', () => {
        localStorage.removeItem('shubh_gemini_api_key');
        localStorage.removeItem('shubh_maps_api_key');
        if (keyInput) keyInput.value = '';
        const mapsInput = document.getElementById('user-maps-key-input');
        if (mapsInput) mapsInput.value = '';
        const msg = document.getElementById('key-status-msg');
        if (msg) {
          msg.textContent = 'API keys cleared from browser memory.';
          msg.className = 'key-status-msg';
          msg.classList.remove('hidden');
        }
        updateKeyDotStatus();
      });
    }

    if (apikeyForm) {
      apikeyForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const geminiVal = keyInput ? keyInput.value.trim() : '';
        const mapsInput = document.getElementById('user-maps-key-input');
        const mapsVal = mapsInput ? mapsInput.value.trim() : '';
        const msg = document.getElementById('key-status-msg');

        if (geminiVal) {
          localStorage.setItem('shubh_gemini_api_key', geminiVal);
        } else {
          localStorage.removeItem('shubh_gemini_api_key');
        }

        if (mapsVal) {
          localStorage.setItem('shubh_maps_api_key', mapsVal);
        } else {
          localStorage.removeItem('shubh_maps_api_key');
        }

        if (msg) {
          msg.textContent = '✅ API Keys saved cleanly to browser memory!';
          msg.className = 'key-status-msg success';
          msg.classList.remove('hidden');
        }
        updateKeyDotStatus();
        setTimeout(() => {
          if (apikeyModal) apikeyModal.classList.remove('open');
        }, 1200);
      });
    }
  }

  function updateKeyDotStatus() {
    const dot = document.getElementById('key-status-dot');
    const hasClientKey = !!localStorage.getItem('shubh_gemini_api_key');
    if (dot) {
      if (hasClientKey || state.hasServerAPIKey) {
        dot.classList.add('active');
        dot.title = 'API Key Active';
      } else {
        dot.classList.remove('active');
        dot.title = 'API Key Unconfigured';
      }
    }
  }

  // ---------------------------------------------------------------------------
  // 4. Event Profile Modal Controller
  // ---------------------------------------------------------------------------
  function formatDateForInput(dateStr) {
    if (!dateStr) return '';
    if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr;
    const d = new Date(dateStr);
    if (!isNaN(d.getTime())) {
      const year = d.getFullYear();
      const month = String(d.getMonth() + 1).padStart(2, '0');
      const day = String(d.getDate()).padStart(2, '0');
      return `${year}-${month}-${day}`;
    }
    return '';
  }

  async function openEventProfileModal() {
    const modal = document.getElementById('event-profile-modal');
    if (!modal) return;
    try {
      const res = await fetch('/api/event');
      if (res.ok) {
        const evt = await res.json();
        state.event = evt;
        document.getElementById('profile-title-input').value = evt.title || '';
        document.getElementById('profile-type-input').value = evt.eventType || '';
        document.getElementById('profile-hosts-input').value = evt.hostNames || '';
        document.getElementById('profile-date-input').value = formatDateForInput(evt.date);
        document.getElementById('profile-venue-input').value = evt.venue || '';
        document.getElementById('profile-location-input').value = evt.location || '';
        document.getElementById('profile-theme-input').value = evt.aestheticTheme || '';
        document.getElementById('profile-guests-input').value = evt.targetGuestCount || '';
        document.getElementById('profile-desc-input').value = evt.description || '';
      }
    } catch (e) {
      console.warn('Load event profile error:', e);
    }

    const msg = document.getElementById('profile-status-msg');
    if (msg) msg.classList.add('hidden');
    modal.classList.add('open');
  }

  function initEventProfileModal() {
    const modal = document.getElementById('event-profile-modal');
    const closeBtn = document.getElementById('close-event-profile-modal-btn');
    const cancelBtn = document.getElementById('cancel-event-profile-btn');
    const form = document.getElementById('event-profile-form');

    if (closeBtn && modal) closeBtn.addEventListener('click', () => modal.classList.remove('open'));
    if (cancelBtn && modal) cancelBtn.addEventListener('click', () => modal.classList.remove('open'));

    if (form) {
      form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const title = document.getElementById('profile-title-input').value;
        const eventType = document.getElementById('profile-type-input').value;
        const hostNames = document.getElementById('profile-hosts-input').value;
        const date = document.getElementById('profile-date-input').value;
        const venue = document.getElementById('profile-venue-input').value;
        const location = document.getElementById('profile-location-input').value;
        const aestheticTheme = document.getElementById('profile-theme-input').value;
        const targetGuestCount = parseInt(document.getElementById('profile-guests-input').value) || 150;
        const description = document.getElementById('profile-desc-input').value;
        const msg = document.getElementById('profile-status-msg');

        try {
          const res = await fetch('/api/event', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              title, eventType, hostNames, date, venue, location, aestheticTheme, targetGuestCount, description
            })
          });
          if (res.ok) {
            const data = await res.json();
            state.event = data.event || data;
            renderEventProfile();
            if (msg) {
              msg.textContent = '✅ Event profile updated successfully!';
              msg.className = 'key-status-msg success';
              msg.classList.remove('hidden');
            }
            setTimeout(() => {
              if (modal) modal.classList.remove('open');
            }, 1000);
          }
        } catch (err) {
          console.error('Update event profile error:', err);
        }
      });
    }
  }

  // ---------------------------------------------------------------------------
  // 5. Account Settings Modal Controller & Account Deletion
  // ---------------------------------------------------------------------------
  function openAccountSettingsModal() {
    const modal = document.getElementById('account-settings-modal');
    if (!modal) return;
    const u = state.user || {};
    document.getElementById('setting-user-id').textContent = u.userId || u.id || '-';
    document.getElementById('setting-user-email').textContent = u.userEmail || u.email || '-';
    document.getElementById('setting-user-role').textContent = (u.userRole || u.role || 'Planner').toUpperCase();
    
    const keyStatus = document.getElementById('setting-key-status');
    const hasKey = !!localStorage.getItem('shubh_gemini_api_key') || state.hasServerAPIKey;
    if (keyStatus) {
      keyStatus.textContent = hasKey ? 'Active API Key' : 'Unconfigured';
      keyStatus.style.color = hasKey ? '#34D399' : '#F87171';
    }
    modal.classList.add('open');
  }

  function initAccountSettingsModal() {
    const modal = document.getElementById('account-settings-modal');
    const closeBtn = document.getElementById('close-account-settings-modal-btn');
    const openApiKeyBtn = document.getElementById('settings-open-apikey-btn');
    const deleteBtn = document.getElementById('delete-account-btn');

    if (closeBtn && modal) closeBtn.addEventListener('click', () => modal.classList.remove('open'));
    if (openApiKeyBtn) {
      openApiKeyBtn.addEventListener('click', () => {
        if (modal) modal.classList.remove('open');
        const keyModal = document.getElementById('apikey-modal');
        if (keyModal) keyModal.classList.add('open');
      });
    }

    if (deleteBtn) {
      deleteBtn.addEventListener('click', async () => {
        if (!confirm('⚠️ Are you sure you want to permanently delete your account? This action cannot be undone.')) {
          return;
        }

        try {
          const res = await fetch('/api/auth/account', { method: 'DELETE' });
          if (res.ok) {
            alert('Your account has been deleted.');
            localStorage.removeItem('shubh_session_token');
            localStorage.removeItem('shubh_gemini_api_key');
            state.user = null;
            if (modal) modal.classList.remove('open');
            updateUserUI();
          } else {
            alert('Account deletion failed. Please try again.');
          }
        } catch (err) {
          console.error('Account deletion error:', err);
        }
      });
    }
  }

  // ---------------------------------------------------------------------------
  // 6. Tab Navigation
  // ---------------------------------------------------------------------------
  function initTabs() {
    const tabBtns = document.querySelectorAll('.tab-btn');
    const tabPanes = document.querySelectorAll('.tab-pane');

    tabBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const targetTab = btn.getAttribute('data-tab');

        tabBtns.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        tabPanes.forEach(pane => {
          if (pane.id === targetTab) {
            pane.classList.add('active');
          } else {
            pane.classList.remove('active');
          }
        });

        state.activeTab = targetTab;
      });
    });
  }

  // ---------------------------------------------------------------------------
  // 7. AI Assistant Chat Controller (SSE Real-Time Streaming)
  // ---------------------------------------------------------------------------
  function initChat() {
    const chatForm = document.getElementById('chat-form');
    const chatInput = document.getElementById('chat-input');
    const promptChips = document.querySelectorAll('.prompt-chip');
    const clearChatBtn = document.getElementById('clear-chat-btn');

    if (clearChatBtn) {
      clearChatBtn.addEventListener('click', () => {
        const chatMessages = document.getElementById('chat-messages');
        if (chatMessages) {
          chatMessages.innerHTML = `
            <div class="message message-system">
              <div class="msg-bubble">
                Chat cleared. Tell me your celebration details and I will configure your event workspace!
              </div>
            </div>
          `;
        }
        state.sessionId = '';
      });
    }

    if (promptChips) {
      promptChips.forEach(chip => {
        chip.addEventListener('click', () => {
          const command = chip.getAttribute('data-command') || chip.getAttribute('data-prompt');
          if (command) {
            sendChatMessage(command);
          }
        });
      });
    }

    // Slash Commands Autocomplete Menu Controller
    const slashMenu = document.getElementById('slash-commands-menu');
    let activeCmdIndex = -1;

    if (chatInput && slashMenu) {
      const commandItems = Array.from(slashMenu.querySelectorAll('.slash-command-item'));

      const hideSlashMenu = () => {
        slashMenu.classList.add('hidden');
        activeCmdIndex = -1;
        commandItems.forEach(item => item.classList.remove('active'));
      };

      const filterSlashCommands = (query) => {
        const term = query.toLowerCase().trim();
        let visibleCount = 0;

        commandItems.forEach(item => {
          const cmd = item.getAttribute('data-cmd') || '';
          if (cmd.toLowerCase().includes(term) || term === '/') {
            item.style.display = 'flex';
            visibleCount++;
          } else {
            item.style.display = 'none';
          }
        });

        if (visibleCount > 0) {
          slashMenu.classList.remove('hidden');
        } else {
          hideSlashMenu();
        }
      };

      chatInput.addEventListener('input', (e) => {
        const val = e.target.value;
        if (val.startsWith('/')) {
          filterSlashCommands(val);
        } else {
          hideSlashMenu();
        }
      });

      chatInput.addEventListener('keydown', (e) => {
        if (slashMenu.classList.contains('hidden')) return;

        const visibleItems = commandItems.filter(item => item.style.display !== 'none');
        if (visibleItems.length === 0) return;

        if (e.key === 'ArrowDown') {
          e.preventDefault();
          activeCmdIndex = (activeCmdIndex + 1) % visibleItems.length;
          visibleItems.forEach((item, idx) => {
            item.classList.toggle('active', idx === activeCmdIndex);
          });
        } else if (e.key === 'ArrowUp') {
          e.preventDefault();
          activeCmdIndex = (activeCmdIndex - 1 + visibleItems.length) % visibleItems.length;
          visibleItems.forEach((item, idx) => {
            item.classList.toggle('active', idx === activeCmdIndex);
          });
        } else if (e.key === 'Enter' || e.key === 'Tab') {
          if (activeCmdIndex >= 0 && activeCmdIndex < visibleItems.length) {
            e.preventDefault();
            const selectedCmd = visibleItems[activeCmdIndex].getAttribute('data-cmd');
            chatInput.value = selectedCmd;
            hideSlashMenu();
            sendChatMessage(selectedCmd);
          }
        } else if (e.key === 'Escape') {
          hideSlashMenu();
        }
      });

      commandItems.forEach(item => {
        item.addEventListener('click', () => {
          const selectedCmd = item.getAttribute('data-cmd');
          if (selectedCmd) {
            chatInput.value = selectedCmd;
            hideSlashMenu();
            sendChatMessage(selectedCmd);
          }
        });
      });

      document.addEventListener('click', (e) => {
        if (chatForm && !chatForm.contains(e.target)) {
          hideSlashMenu();
        }
      });
    }

    if (chatForm) {
      chatForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const slashMenu = document.getElementById('slash-commands-menu');
        if (slashMenu) slashMenu.classList.add('hidden');
        const text = chatInput ? chatInput.value.trim() : '';
        if (text) {
          sendChatMessage(text);
        }
      });
    }

    const chatMessages = document.getElementById('chat-messages');
    if (chatMessages) {
      chatMessages.addEventListener('click', (e) => {
        const img = e.target.closest('img') || e.target.closest('.chat-card-preview-box')?.querySelector('img');
        if (img && img.src) {
          openCardPreview(img.src, 'AI Invitation Artwork', 'Generated Concept');
        }
      });
    }
  }

  function sendChatMessage(userText) {
    const chatInput = document.getElementById('chat-input');
    const chatMessages = document.getElementById('chat-messages');
    const sendBtn = document.getElementById('chat-send-btn');
    if (!chatMessages) return;

    if (chatInput) chatInput.value = '';

    const userMsgDiv = document.createElement('div');
    userMsgDiv.className = 'message message-user';
    userMsgDiv.innerHTML = `<div class="msg-bubble">${escapeHtml(userText)}</div>`;
    chatMessages.appendChild(userMsgDiv);

    const botMsgDiv = document.createElement('div');
    botMsgDiv.className = 'message message-assistant';
    const botBubble = document.createElement('div');
    botBubble.className = 'msg-bubble markdown-body';
    botBubble.innerHTML = '<div class="thinking-status-box"><span class="typing-indicator"><span></span><span></span><span></span></span> <span>AI Assistant is orchestrating tools & processing...</span></div>';
    botMsgDiv.appendChild(botBubble);
    chatMessages.appendChild(botMsgDiv);

    chatMessages.scrollTop = chatMessages.scrollHeight;

    if (sendBtn) {
      sendBtn.disabled = true;
      sendBtn.innerHTML = '<span>⏳ Thinking...</span>';
    }

    const resetSendBtn = () => {
      if (sendBtn) {
        sendBtn.disabled = false;
        sendBtn.innerHTML = '<span>Send</span> <span>➔</span>';
      }
    };

    let accumulatedText = '';
    const clientApiKey = localStorage.getItem('shubh_gemini_api_key') || '';
    const clientMapsKey = localStorage.getItem('shubh_maps_api_key') || '';
    const url = `/api/stream/assistant?message=${encodeURIComponent(userText)}&sessionId=${encodeURIComponent(state.sessionId)}&apiKey=${encodeURIComponent(clientApiKey)}&mapsKey=${encodeURIComponent(clientMapsKey)}`;
    const eventSource = new EventSource(url);

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.sessionId) state.sessionId = data.sessionId;

        if (data.token) {
          accumulatedText += data.token;
          botBubble.innerHTML = formatMarkdownText(accumulatedText);
          if (window.ChatWidgets) {
            window.ChatWidgets.mountAll(botBubble, {
              onSuccess: () => loadAllData()
            });
          }
          chatMessages.scrollTop = chatMessages.scrollHeight;
        }

        if (data.done) {
          eventSource.close();
          resetSendBtn();
          if (window.ChatWidgets) {
            window.ChatWidgets.mountAll(botBubble, {
              onSuccess: () => loadAllData()
            });
          }
          loadAllData();
          setTimeout(() => {
            chatMessages.scrollTop = chatMessages.scrollHeight;
            const widget = botBubble.querySelector('.chat-widget-card');
            if (widget) {
              widget.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            }
          }, 60);
        }
      } catch (err) {
        console.error('SSE parse error:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.warn('SSE stream error:', err);
      eventSource.close();
      resetSendBtn();
      if (!accumulatedText) {
        botBubble.innerHTML = '<span class="text-danger">⚠️ Assistant error or API Key required. Please check your Gemini API key settings.</span>';
      }
    };
  }

  // ---------------------------------------------------------------------------
  // 8. Invitation Studio Controller
  // ---------------------------------------------------------------------------
  async function fetchDesigns() {
    try {
      const res = await fetch('/api/designs');
      if (res.ok) {
        state.designs = await res.json();
        renderDesignGrid();
      }
    } catch (e) {
      console.warn('Fetch designs error:', e);
    }
  }

  function initStudio() {
    const studioForm = document.getElementById('design-form');
    const suggestionsBtn = document.getElementById('generate-suggestions-btn');
    const aspectOptions = document.querySelectorAll('.aspect-option');
    const chipBtns = document.querySelectorAll('#custom-elements-chips .chip');
    const colorSwatches = document.querySelectorAll('.swatch');
    const typoPreset = document.getElementById('typography-preset');

    aspectOptions.forEach(opt => {
      opt.addEventListener('click', () => {
        aspectOptions.forEach(o => o.classList.remove('active'));
        opt.classList.add('active');
        const ratio = opt.getAttribute('data-ratio');
        const ratioInput = document.getElementById('aspect-ratio-input');
        if (ratioInput) ratioInput.value = ratio;
      });
    });

    chipBtns.forEach(chip => {
      chip.addEventListener('click', () => {
        chip.classList.toggle('chip-selected');
      });
    });

    const addElementBtn = document.getElementById('add-custom-element-btn');
    const newElementInput = document.getElementById('new-custom-element-input');
    if (addElementBtn && newElementInput) {
      const handleAdd = () => {
        const val = newElementInput.value.trim();
        if (!val) return;

        const chipsContainer = document.getElementById('custom-elements-chips');
        if (chipsContainer) {
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.className = 'chip chip-selected';
          btn.setAttribute('data-value', val);
          btn.textContent = `✨ ${val}`;
          btn.addEventListener('click', () => {
            btn.classList.toggle('chip-selected');
          });
          chipsContainer.appendChild(btn);
          newElementInput.value = '';
        }
      };

      addElementBtn.addEventListener('click', handleAdd);
      newElementInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          handleAdd();
        }
      });
    }

    colorSwatches.forEach(swatch => {
      swatch.addEventListener('click', () => {
        const color = swatch.getAttribute('data-color');
        const input = document.getElementById('primary-color-input');
        if (input) input.value = color;
      });
    });

    if (typoPreset) {
      typoPreset.addEventListener('change', (e) => {
        const customIn = document.getElementById('typography-input');
        if (customIn && e.target.value !== 'custom') {
          customIn.value = e.target.value;
        }
      });
    }

    const generatePromptSuggestions = async () => {
      const aspect = document.getElementById('aspect-ratio-input')?.value || '9:16';
      const theme = document.getElementById('aesthetic-theme-input')?.value || 'Clay 3D';
      let elements = Array.from(document.querySelectorAll('#custom-elements-chips .chip-selected'))
        .map(chip => chip.getAttribute('data-value') || chip.textContent.replace(/[^\w\s]/gi, '').trim())
        .filter(Boolean);
      if (!elements || elements.length === 0) {
        elements = ['Elephant Motif', 'Marigold Garlands'];
      }
      const typography = document.getElementById('typography-input')?.value || 'Cinzel Decorative & Outfit';
      const primaryColor = document.getElementById('primary-color-input')?.value || '#D4AF37';
      const specialInstructions = document.getElementById('special-instructions-input')?.value || '';
      const suggestionsGrid = document.getElementById('prompt-suggestions-grid');

      if (suggestionsGrid) {
        suggestionsGrid.innerHTML = '<div class="loading-box" style="text-align:center; padding:30px;"><span class="typing-indicator">✨ Synthesizing AI prompt suggestions...</span></div>';
      }

      try {
        const res = await fetch('/api/flows/invitationPromptSuggestionsFlow', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            aspectRatio: aspect,
            stylePreset: theme,
            customElements: elements,
            typography: typography,
            primaryColor: primaryColor,
            specialInstructions: specialInstructions
          })
        });
        const data = await res.json();
        if (res.ok && data.suggestions && suggestionsGrid) {
          suggestionsGrid.innerHTML = data.suggestions.map((s, idx) => `
            <div class="prompt-suggestion-card" style="background:rgba(255,255,255,0.03); border:1px solid rgba(255,255,255,0.08); padding:16px; border-radius:12px; margin-bottom:12px;">
              <h4 style="color:#D4AF37; font-weight:700; margin-bottom:6px;">Concept ${idx + 1}: ${escapeHtml(s.title || 'Design Prompt')}</h4>
              <p style="font-size:0.85rem; color:#a0aec0; margin-bottom:12px;">${escapeHtml(s.promptText || s)}</p>
              <button type="button" class="btn btn-sm btn-accent generate-art-btn" data-prompt="${escapeHtml(s.promptText || s)}" data-theme="${escapeHtml(theme)}">🎨 Generate Artwork</button>
            </div>
          `).join('');

          document.querySelectorAll('.generate-art-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
              const prompt = btn.getAttribute('data-prompt');
              const styleTheme = btn.getAttribute('data-theme');
              const aspectRatio = document.getElementById('aspect-ratio-input')?.value || '4:5';
              btn.disabled = true;
              btn.classList.add('btn-rendering-active');
              btn.innerHTML = '⏳ Synthesizing & Rendering PNG...';
              try {
                const genRes = await fetch('/api/flows/invitationGeneratorFlow', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    promptText: prompt,
                    prompt: prompt,
                    aestheticTheme: styleTheme,
                    styleTheme: styleTheme,
                    aspectRatio: aspectRatio
                  })
                });
                if (genRes.ok) {
                  const suggestionsGrid = document.getElementById('prompt-suggestions-grid');
                  if (suggestionsGrid) suggestionsGrid.innerHTML = '';
                  await fetchDesigns();
                  setTimeout(() => {
                    const grid = document.getElementById('design-grid');
                    if (grid) {
                      grid.scrollIntoView({ behavior: 'smooth', block: 'start' });
                    }
                  }, 80);
                }
              } catch (err) {
                console.error('Generate art error:', err);
              } finally {
                btn.disabled = false;
                btn.classList.remove('btn-rendering-active');
                btn.innerHTML = '🎨 Generate Artwork';
              }
            });
          });
        }
      } catch (e) {
        console.error('Prompt suggestions error:', e);
      }
    };

    if (studioForm) {
      studioForm.addEventListener('submit', (e) => {
        e.preventDefault();
        generatePromptSuggestions();
      });
    }

    const closePreviewBtn = document.getElementById('close-preview-card-modal-btn');
    if (closePreviewBtn) {
      closePreviewBtn.addEventListener('click', () => {
        const modal = document.getElementById('preview-card-modal');
        if (modal) modal.classList.remove('open');
      });
    }
  }

  let isStudioExpanded = false;

  window.openCardPreview = function(imgUrl, title, theme) {
    const modal = document.getElementById('preview-card-modal');
    const modalImg = document.getElementById('preview-modal-img');
    const modalTitle = document.getElementById('preview-modal-title');
    const modalTheme = document.getElementById('preview-modal-theme');
    const downloadBtn = document.getElementById('preview-modal-download-btn');

    const src = imgUrl || '/assets/generated_card_concept.png';
    if (modalImg) modalImg.src = src;
    if (modalTitle) modalTitle.textContent = title || 'AI Invitation Artwork';
    if (modalTheme) modalTheme.textContent = theme || 'Luxury Concept';
    if (downloadBtn) {
      downloadBtn.href = src;
      downloadBtn.setAttribute('download', `${(title || 'invitation_card').replace(/\s+/g, '_').toLowerCase()}.png`);
    }
    if (modal) modal.classList.add('open');
  };

  function renderDesignGrid() {
    const grid = document.getElementById('design-grid');
    const badge = document.getElementById('designs-count-badge');
    const wrapper = document.getElementById('view-more-wrapper');
    const btnText = document.getElementById('view-more-btn-text');
    const toggleBtn = document.getElementById('view-more-designs-btn');

    const designList = state.designs || [];
    if (badge) badge.textContent = `${designList.length} Concepts`;

    if (designList.length === 0) {
      grid.innerHTML = `
        <div class="empty-state-box" style="grid-column: 1 / -1; text-align: center; padding: 45px 20px; background: rgba(255,255,255,0.02); border-radius: 12px; border: 1px dashed rgba(255,255,255,0.1);">
          <span class="empty-icon" style="font-size: 2.5rem; display: block; margin-bottom: 8px;">🎨</span>
          <p style="color: #a0aec0;">No AI invitations generated yet. Select an aesthetic theme above and click <strong>"Generate Luxury Invitation Card"</strong>!</p>
        </div>
      `;
      if (wrapper) wrapper.classList.add('hidden');
      return;
    }

    const visibleDesigns = isStudioExpanded ? designList : designList.slice(0, 2);

    grid.innerHTML = visibleDesigns.map(d => {
      const imgUrl = escapeHtml(d.imageUrl || '/assets/card_1787721714508660800.png');
      const theme = escapeHtml(d.styleTheme || 'Luxury Concept');
      const title = escapeHtml(d.headline || d.title || 'Aarav\'s Naming Ceremony');
      const dateStr = escapeHtml(d.dateStr || 'October 12, 2026');
      const venueStr = escapeHtml(d.venueStr || 'Marhaba Mini Function Hall');
      const aspect = escapeHtml(d.aspectRatio || '4:5');
      const fullPrompt = d.prompt || '';

      return `
        <div class="card-art-box" id="design-card-${d.id}" style="background: rgba(15, 23, 42, 0.6); border: 1px solid rgba(255, 255, 255, 0.08); border-radius: 12px; overflow: hidden; display: flex; flex-direction: column; justify-content: space-between;">
          <div class="card-img-wrapper" data-img="${imgUrl}" data-title="${title}" data-theme="${theme}" style="width: 100%; height: 380px; background: #050811; cursor: pointer; position: relative;" title="Click for full-screen preview">
            <img src="${imgUrl}" alt="${title}" class="card-img" style="width: 100%; height: 100%; object-fit: contain; display: block;" loading="lazy">
          </div>
          
          <div class="card-body" style="padding: 16px; display: flex; flex-direction: column; gap: 6px;">
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <h4 style="font-family: var(--font-display); font-weight: 700; color: #D4AF37; font-size: 1.1rem; margin: 0;">${theme}</h4>
              <span class="badge badge-accent" style="font-size: 0.75rem; padding: 2px 8px; border-radius: 10px; background: rgba(6, 182, 212, 0.15); color: #06B6D4; border: 1px solid rgba(6, 182, 212, 0.3); font-weight: 700;">${aspect}</span>
            </div>

            <div style="font-size: 0.82rem; color: #94A3B8; font-weight: 500;">
              ${title} • ${dateStr} • ${venueStr}
            </div>

            <p style="font-size: 0.82rem; color: #CBD5E1; line-height: 1.45; margin: 4px 0 10px 0; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; text-overflow: ellipsis;">
              ${escapeHtml(fullPrompt)}
            </p>

            <div style="display: flex; justify-content: space-between; align-items: center; gap: 10px; margin-top: auto;">
              <a href="${imgUrl}" download="invitation_card.png" class="btn btn-outline btn-sm" style="font-size: 0.78rem; font-weight: 600; padding: 6px 16px; border-radius: 20px; text-decoration: none; display: inline-flex; align-items: center; gap: 6px; border: 1px solid rgba(255,255,255,0.15); color: #FFFFFF; background: rgba(255,255,255,0.05);">
                🤖 Download PNG
              </a>
              <button type="button" class="btn-prompt-copy btn btn-sm" data-prompt="${escapeHtml(fullPrompt)}" style="font-size: 0.78rem; font-weight: 700; padding: 6px 16px; border-radius: 20px; display: inline-flex; align-items: center; gap: 6px; border: none; color: #0F172A; background: #FFFFFF; cursor: pointer;">
                📋 Copy Prompt
              </button>
            </div>
          </div>
        </div>
      `;
    }).join('');

    // Bind Image Click Listeners for Full-Screen Preview Modal
    grid.querySelectorAll('.card-img-wrapper').forEach(wrapper => {
      wrapper.onclick = () => {
        const img = wrapper.getAttribute('data-img');
        const title = wrapper.getAttribute('data-title');
        const theme = wrapper.getAttribute('data-theme');
        openCardPreview(img, title, theme);
      };
    });

    // Toggle button visibility outside design-grid
    if (wrapper && toggleBtn) {
      if (state.designs.length > 2) {
        wrapper.classList.remove('hidden');
        wrapper.style.display = 'block';
        if (btnText) {
          btnText.textContent = isStudioExpanded ? '▲ Show Less' : `👁️ View All ${state.designs.length} Designs`;
        }
        toggleBtn.onclick = (e) => {
          e.stopPropagation();
          isStudioExpanded = !isStudioExpanded;
          renderDesignGrid();
        };
      } else {
        wrapper.classList.add('hidden');
        wrapper.style.display = 'none';
      }
    }

    // Bind Copy Prompt Listeners
    setTimeout(() => {
      document.querySelectorAll('.btn-prompt-copy').forEach(btn => {
        btn.onclick = (e) => {
          e.stopPropagation();
          const textToCopy = btn.getAttribute('data-prompt');
          if (navigator.clipboard && textToCopy) {
            navigator.clipboard.writeText(textToCopy);
            const orig = btn.innerHTML;
            btn.innerHTML = '✅ Copied!';
            btn.style.background = '#34D399';
            btn.style.color = '#070A14';
            setTimeout(() => {
              btn.innerHTML = orig;
              btn.style.background = '#FFFFFF';
              btn.style.color = '#0F172A';
            }, 1800);
          }
        };
      });
    }, 30);
  }

  // ---------------------------------------------------------------------------
  // 9. Guest Roster & RSVPs Controller
  // ---------------------------------------------------------------------------
  async function fetchGuests() {
    try {
      const res = await fetch('/api/guests');
      if (res.ok) {
        state.guests = await res.json();
        renderGuestStats();
        renderGuestsTable();
      }
    } catch (e) {
      console.warn('Fetch guests error:', e);
    }
  }

  function initRoster() {
    const searchInput = document.getElementById('guest-search');
    const filterSelect = document.getElementById('guest-status-filter');
    const quickForm = document.getElementById('quick-guest-form');

    if (searchInput) searchInput.addEventListener('input', renderGuestsTable);
    if (filterSelect) filterSelect.addEventListener('change', renderGuestsTable);

    if (quickForm) {
      quickForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const name = document.getElementById('quick-guest-name').value;
        const category = document.getElementById('quick-guest-category').value;
        const rsvpStatus = document.getElementById('quick-guest-status').value;
        const plusOnes = parseInt(document.getElementById('quick-guest-plusones').value) || 0;
        const dietaryRequirements = document.getElementById('quick-guest-diet').value;

        try {
          const res = await fetch('/api/guests', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, category, rsvpStatus, plusOnes, dietaryRequirements })
          });
          if (res.ok) {
            quickForm.reset();
            await fetchGuests();
          }
        } catch (err) {
          console.error('Quick add guest error:', err);
        }
      });
    }
  }

  function renderGuestStats() {
    const totalEl = document.getElementById('stat-total');
    const confEl = document.getElementById('stat-confirmed');
    const pendEl = document.getElementById('stat-pending');
    const declEl = document.getElementById('stat-declined');

    let conf = 0, pend = 0, decl = 0;
    state.guests.forEach(g => {
      const st = (g.rsvpStatus || '').toLowerCase();
      if (st === 'confirmed' || st === 'attending') conf++;
      else if (st === 'declined') decl++;
      else pend++;
    });

    if (totalEl) totalEl.textContent = state.guests.length;
    if (confEl) confEl.textContent = conf;
    if (pendEl) pendEl.textContent = pend;
    if (declEl) declEl.textContent = decl;
  }

  function renderGuestsTable() {
    const tbody = document.getElementById('guests-table-body');
    const searchInput = document.getElementById('guest-search');
    const filterSelect = document.getElementById('guest-status-filter');

    if (!tbody) return;

    const query = searchInput ? searchInput.value.toLowerCase().trim() : '';
    const statusFilter = filterSelect ? filterSelect.value.toLowerCase() : 'all';

    const filtered = state.guests.filter(g => {
      const text = `${g.name} ${g.category} ${g.phone} ${g.notes} ${g.dietaryRequirements}`.toLowerCase();
      const matchesQuery = !query || text.includes(query);
      const st = (g.rsvpStatus || 'pending').toLowerCase();
      const matchesStatus = statusFilter === 'all' || st === statusFilter;
      return matchesQuery && matchesStatus;
    });

    if (filtered.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="6" class="empty-table-msg" style="text-align:center; padding:30px; color:#a0aec0;">
            No matching guests found.
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = filtered.map(g => {
      const st = (g.rsvpStatus || 'pending').toLowerCase();
      let badgeClass = 'tag-pending';
      if (st === 'confirmed' || st === 'attending') badgeClass = 'tag-confirmed';
      if (st === 'declined') badgeClass = 'tag-declined';

      return `
        <tr id="guest-row-${g.id}" class="guest-row">
          <td style="font-weight:600;">${escapeHtml(g.name)}</td>
          <td style="color:var(--text-secondary);">${escapeHtml(g.category || 'Family')}</td>
          <td>
            <span class="status-tag ${badgeClass}">${escapeHtml((g.rsvpStatus || 'Pending').toUpperCase())}</span>
          </td>
          <td class="text-secondary">+${g.plusOnes || 0} (${g.plusOnes || 0} plus ones)</td>
          <td>
            <span class="diet-tag" style="background:rgba(212,175,55,0.12); color:#F3E5AB; padding:2px 8px; border-radius:12px; font-size:0.8rem; border:1px solid rgba(212,175,55,0.3);">${escapeHtml(g.dietaryRequirements || 'No Restrictions')}</span>
          </td>
          <td class="action-cell">
            <div class="row-actions" style="display:flex; gap:6px;">
              <button type="button" class="btn-icon toggle-rsvp-btn" data-id="${g.id}" title="Toggle RSVP Status" style="background:rgba(255,255,255,0.06); border:1px solid rgba(255,255,255,0.1); border-radius:6px; padding:4px 8px; cursor:pointer;">🔄</button>
              <button type="button" class="btn-icon delete-guest-btn text-danger" data-id="${g.id}" data-name="${escapeHtml(g.name)}" title="Remove Guest" style="background:rgba(239,68,68,0.1); border:1px solid rgba(239,68,68,0.3); border-radius:6px; padding:4px 8px; cursor:pointer;">🗑️</button>
            </div>
          </td>
        </tr>
      `;
    }).join('');

    tbody.querySelectorAll('.toggle-rsvp-btn').forEach(btn => {
      btn.addEventListener('click', async () => {
        const id = btn.getAttribute('data-id');
        try {
          const res = await fetch(`/api/guests/${id}/rsvp`, { method: 'POST' });
          if (res.ok) await fetchGuests();
        } catch (e) {
          console.error('Toggle RSVP error:', e);
        }
      });
    });

    tbody.querySelectorAll('.delete-guest-btn').forEach(btn => {
      btn.addEventListener('click', async () => {
        const id = btn.getAttribute('data-id');
        const name = btn.getAttribute('data-name');
        if (!confirm(`Are you sure you want to remove ${name} from the guest roster?`)) return;
        try {
          const res = await fetch(`/api/guests/${id}`, { method: 'DELETE' });
          if (res.ok) await fetchGuests();
        } catch (e) {
          console.error('Delete guest error:', e);
        }
      });
    });
  }

  // ---------------------------------------------------------------------------
  // 10. Itinerary Controller
  // ---------------------------------------------------------------------------
  async function fetchItinerary() {
    try {
      const res = await fetch('/api/itinerary');
      if (res.ok) {
        state.itinerary = await res.json();
        renderItineraryTimeline();
      }
    } catch (e) {
      console.warn('Fetch itinerary error:', e);
    }
  }

  function initItinerary() {
    const form = document.getElementById('add-session-form');
    if (form) {
      form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const time = document.getElementById('session-time').value;
        const title = document.getElementById('session-title').value;
        const location = document.getElementById('session-location').value;
        const description = document.getElementById('session-desc').value;

        try {
          const res = await fetch('/api/itinerary', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ time, title, location, description })
          });
          if (res.ok) {
            form.reset();
            await fetchItinerary();
          }
        } catch (err) {
          console.error('Add itinerary error:', err);
        }
      });
    }
  }

  function renderItineraryTimeline() {
    const timeline = document.getElementById('itinerary-timeline');
    if (!timeline) return;

    if (!state.itinerary || state.itinerary.length === 0) {
      timeline.innerHTML = `
        <div class="empty-state-box" style="text-align:center; padding:30px; color:#a0aec0;">
          <span class="empty-icon" style="font-size:2rem; display:block; margin-bottom:8px;">📅</span>
          No itinerary items scheduled yet. Add a session on the right!
        </div>
      `;
      return;
    }

    timeline.innerHTML = `
      <div class="timeline-wrapper" style="display:flex; flex-direction:column; gap:16px;">
        ${state.itinerary.map(item => `
          <div class="timeline-item" id="itinerary-item-${item.id}" style="background:rgba(255,255,255,0.03); border:1px solid rgba(255,255,255,0.08); padding:16px; border-radius:12px; display:flex; gap:16px; align-items:flex-start;">
            <div class="timeline-time" style="font-weight:700; color:#D4AF37; min-width:90px; font-size:0.95rem;">${escapeHtml(item.time)}</div>
            <div class="timeline-content">
              <h4 class="timeline-title" style="font-weight:700; margin-bottom:4px; font-size:1rem;">${escapeHtml(item.title)}</h4>
              <p class="timeline-desc" style="font-size:0.85rem; color:#a0aec0; margin-bottom:6px; line-height:1.4;">${escapeHtml(item.description)}</p>
              ${item.location ? `<span class="timeline-loc" style="font-size:0.8rem; color:#00D4FF;">📍 ${escapeHtml(item.location)}</span>` : ''}
            </div>
          </div>
        `).join('')}
      </div>
    `;
  }

  // ---------------------------------------------------------------------------
  // Utilities
  // ---------------------------------------------------------------------------
  function formatHumanReadableDate(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    if (!isNaN(d.getTime())) {
      return d.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
    }
    return dateStr;
  }

  function formatMarkdownText(text) {
    if (!text) return '';
    let html = text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');

    html = html
      .replace(/\[WIDGET:ADD_GUEST\]/g, '<div class="widget-mount-add-guest"></div>')
      .replace(/\[WIDGET:GENERATE_INVITATION\]/g, '<div class="widget-mount-generate-invitation"></div>')
      .replace(/\[WIDGET:ADD_ITINERARY\]/g, '<div class="widget-mount-schedule-session"></div>');

    html = html.replace(/!\[(.*?)\]\((.*?)\)/g, (match, alt, url) => {
      const cleanUrl = url.replace(/&amp;/g, '&');
      return `<div class="chat-card-preview-box" style="margin: 12px 0; border-radius: 12px; overflow: hidden; border: 1px solid rgba(212,175,55,0.3); box-shadow: 0 8px 24px rgba(0,0,0,0.4); cursor: pointer; position: relative;" title="Click for full-screen preview">
        <img src="${cleanUrl}" alt="${alt}" class="chat-rendered-img" style="width: 100%; max-height: 400px; object-fit: cover; display: block; cursor: pointer; transition: transform 0.2s ease;" />
        <div style="position: absolute; bottom: 8px; right: 8px; background: rgba(15,23,42,0.85); border: 1px solid rgba(255,255,255,0.15); color: #F8FAFC; padding: 4px 10px; border-radius: 20px; font-size: 0.75rem; pointer-events: none; backdrop-filter: blur(8px);">🔍 Click for Full-Screen Preview</div>
      </div>`;
    });

    html = html.replace(/\[(.*?)\]\((.*?)\)/g, (match, linkText, url) => {
      const cleanUrl = url.replace(/&amp;/g, '&');
      return `<a href="${cleanUrl}" target="_blank" rel="noopener" class="link-accent" style="color:var(--accent-gold, #D4AF37); text-decoration:underline;">${linkText} ↗</a>`;
    });

    html = html
      .replace(/### (.*?)(<br>|\n|$)/g, '<h3 style="margin-top:12px; margin-bottom:6px; font-weight:700; color:var(--accent-gold, #D4AF37);">$1</h3>')
      .replace(/## (.*?)(<br>|\n|$)/g, '<h2 style="margin-top:14px; margin-bottom:8px; font-weight:700; color:var(--accent-gold, #D4AF37);">$1</h2>')
      .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      .replace(/\*(.*?)\*/g, '<em>$1</em>')
      .replace(/`([^`]+)`/g, '<code style="background:rgba(255,255,255,0.1); padding:2px 6px; border-radius:4px; font-family:monospace;">$1</code>')
      .replace(/\n/g, '<br>');

    return html;
  }

  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }
});
