// HTMX Bridge - Intercepts HTMX requests for anonymous users and routes to local storage
(function() {
  // Check if user is authenticated
  function isAuthenticated() {
    return document.body.dataset.userEmail && document.body.dataset.userEmail !== '';
  }

  // Generate HTML for option row (matches server-side template)
  function renderOptionRow(opt, totalWeight) {
    const probability = ((opt.weight / totalWeight) * 100).toFixed(1);
    const weightBorderClass = getWeightBorderClass(opt.weight);
    const durationHTML = opt.duration !== null && opt.duration !== undefined 
      ? `<span class="text-white/70 text-sm">⏱ ${formatDuration(opt.duration)}</span>` 
      : '';
    
    const tagsHTML = opt.tags && opt.tags.length > 0
      ? `<div class="flex gap-1 flex-wrap">
          ${opt.tags.map(tag => `
            <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-purple-500/20 text-purple-200 border border-purple-500/30">
              ${escapeHTML(tag)}
            </span>
          `).join('')}
        </div>`
      : '';

    return `
      <div id="option-${opt.id}" class="bg-white/10 backdrop-blur-sm rounded-lg p-4 border border-white/20 hover:bg-white/20 transition-all ${weightBorderClass}">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3 flex-wrap">
            <span class="text-white font-medium">${escapeHTML(opt.text)}</span>
            ${tagsHTML}
            ${durationHTML}
          </div>
          <div class="flex items-center gap-2">
            <div class="text-blue-200 text-sm">${probability}%</div>
            <button
              data-delete-option="${opt.id}"
              class="p-2 hover:bg-red-500/20 rounded-lg transition-colors text-red-300"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"></path>
              </svg>
            </button>
            <button
              data-expand-option="${opt.id}"
              class="p-2 hover:bg-white/20 rounded-lg transition-colors text-white/70 hover:text-white"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6 chevron-icon">
                <path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    `;
  }

  function getWeightBorderClass(weight) {
    if (weight >= 8) return 'border-l-4 border-l-green-500';
    if (weight >= 5) return 'border-l-4 border-l-yellow-500';
    return 'border-l-4 border-l-red-500';
  }

  function formatDuration(minutes) {
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;
    if (hours > 0) {
      return `${hours}h ${mins}m`;
    }
    return `${mins}m`;
  }

  function escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  // Render all options
  function renderOptionsList(options) {
    if (!options || options.length === 0) {
      return '<div class="text-white/50 text-center py-8">No options yet. Add your first option above!</div>';
    }

    const totalWeight = LocalStorageManager.getTotalWeight(options);
    return options.map(opt => renderOptionRow(opt, totalWeight)).join('');
  }

  // Render result card
  function renderResult(option) {
    const probability = ((option.weight / LocalStorageManager.getTotalWeight(LocalStorageManager.getOptions())) * 100).toFixed(1);
    const durationHTML = option.duration !== null && option.duration !== undefined
      ? `<span class="badge">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"></path>
          </svg>
          ${formatDuration(option.duration)}
        </span>`
      : '';

    return `
      <div class="bg-white/15 backdrop-blur-md rounded-2xl p-8 border border-white/30 shadow-2xl animate-scale-in" id="result-card">
        <div class="text-center space-y-5">
          <div class="text-white/70 text-sm uppercase tracking-wider font-medium">
            🎯 Your Decision:
          </div>
          <div class="text-5xl md:text-6xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-blue-400 via-indigo-400 to-purple-400 animate-subtle-glow py-2">
            ${escapeHTML(option.text)}
          </div>
          <div class="flex justify-center gap-3 flex-wrap">
            ${durationHTML}
            <span class="badge">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347a1.125 1.125 0 0 1-1.667-.986V5.653Z"></path>
              </svg>
              ${probability}%
            </span>
          </div>
          <div class="pt-2">
            <button 
              data-dismiss-result
              class="px-8 py-3 bg-gradient-to-r from-emerald-500 to-green-600 hover:from-emerald-600 hover:to-green-700 text-white font-semibold rounded-xl transition-all shadow-lg hover:shadow-xl hover:scale-105 active:scale-95"
            >
              <span class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5"></path>
                </svg>
                Got it!
              </span>
            </button>
          </div>
        </div>
        <script>
          celebrateDecision();
        </script>
      </div>
    `;
  }

  // Render no options available message
  function renderNoOptions(timeConstraintMinutes) {
    const durationText = timeConstraintMinutes > 0 ? formatDuration(timeConstraintMinutes) : 'any time';
    return `
      <div class="bg-white/15 backdrop-blur-md rounded-2xl p-8 border border-white/30 shadow-2xl animate-scale-in" id="result-card">
        <div class="text-center space-y-5">
          <div class="text-white/70 text-sm uppercase tracking-wider font-medium">
            🤷 No Options Available
          </div>
          <div class="text-3xl md:text-4xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-amber-400 via-orange-400 to-red-400 py-2">
            No options available within ${durationText}
          </div>
          <div class="text-white/70 text-base">
            Try increasing your time constraint or clearing the filter
          </div>
          <div class="pt-2">
            <button 
              data-dismiss-result
              class="px-8 py-3 bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white font-semibold rounded-xl transition-all shadow-lg hover:shadow-xl hover:scale-105 active:scale-95"
            >
              <span class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5"></path>
                </svg>
                Got it!
              </span>
            </button>
          </div>
        </div>
      </div>
    `;
  }

  // Render manage modal
  function renderManageModal() {
    const options = LocalStorageManager.getOptions();
    const totalWeight = LocalStorageManager.getTotalWeight(options);
    
    return `
      <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50" onclick="if(event.target === this) { this.innerHTML = ''; }">
        <div class="bg-white/10 backdrop-blur-md rounded-2xl border border-white/30 max-w-2xl w-full max-h-[80vh] overflow-hidden modal-animate" onclick="event.stopPropagation()">
          <div class="p-6 border-b border-white/20">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-2xl font-bold text-white">Manage Options</h2>
              </div>
              <button
                data-close-modal
                class="text-white/70 hover:text-white text-2xl transition-colors"
              >
                ×
              </button>
            </div>
          </div>
          <div class="p-6 overflow-y-auto max-h-[50vh]">
            <div class="space-y-3" id="options-list">
              ${renderOptionsList(options)}
            </div>
          </div>
          <div class="p-6 border-t border-white/20">
            <div class="flex gap-3 mb-4">
              <form
                data-add-option-form
                class="flex-1 flex flex-col gap-2"
              >
                <div class="flex gap-2">
                  <input
                    type="text"
                    name="text"
                    placeholder="Enter new activity..."
                    required
                    class="flex-1 px-4 py-2 rounded-lg border border-white/20 bg-white/10 text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <button
                    type="submit"
                    class="bg-green-500 hover:bg-green-600 text-white px-6 py-2 rounded-lg transition-colors"
                  >
                    Add
                  </button>
                </div>
                <input
                  type="text"
                  name="tags"
                  placeholder="Tags (comma-separated, max 5)..."
                  maxlength="100"
                  class="px-4 py-2 rounded-lg border border-white/20 bg-white/10 text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-purple-500 text-sm"
                />
              </form>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  // Parse form data to get time constraint in minutes
  function parseTimeConstraint(formData) {
    const hours = parseInt(formData.get('hours') || '0', 10);
    const minutes = parseInt(formData.get('minutes') || '0', 10);
    const totalMinutes = (hours * 60) + minutes;
    return totalMinutes > 0 ? totalMinutes : null;
  }

  // Parse tags from form data
  function parseTags(formData) {
    const tagsString = formData.get('tags');
    if (!tagsString) {
      // Check for tags[] array format (from tag filter)
      const tagsArray = formData.getAll('tags[]');
      return tagsArray.filter(t => t && t.trim()).map(t => t.trim().toLowerCase());
    }
    
    return tagsString
      .split(',')
      .map(tag => tag.trim().toLowerCase())
      .filter(tag => tag.length > 0)
      .slice(0, 5);
  }

  // Set up event delegation for local storage operations
  function setupEventDelegation() {
    // Handle form submissions and button clicks
    // Use capture phase (true) to catch events before stopPropagation
    document.addEventListener('click', function(e) {
      // Close modal
      if (e.target.closest('[data-close-modal]')) {
        e.preventDefault();
        e.stopPropagation();
        const modal = document.getElementById('manage-modal');
        if (modal) modal.innerHTML = '';
        return;
      }

      // Delete option
      if (e.target.closest('[data-delete-option]')) {
        e.preventDefault();
        const btn = e.target.closest('[data-delete-option]');
        const optionId = btn.getAttribute('data-delete-option');
        try {
          LocalStorageManager.deleteOption(optionId);
          const optionsList = document.getElementById('options-list');
          if (optionsList) {
            optionsList.innerHTML = renderOptionsList(LocalStorageManager.getOptions());
          }
        } catch (error) {
          console.error('Failed to delete option:', error);
        }
        return;
      }

      // Dismiss result
      if (e.target.closest('[data-dismiss-result]')) {
        e.preventDefault();
        const card = document.getElementById('result-card');
        if (card) {
          card.classList.add('animate-fade-out');
          setTimeout(() => card.remove(), 300);
        }
        return;
      }
    }, true); // Capture phase

    // Handle add option form
    document.addEventListener('submit', function(e) {
      if (e.target.hasAttribute('data-add-option-form')) {
        e.preventDefault();
        const formData = new FormData(e.target);
        const text = formData.get('text');
        const tags = parseTags(formData);

        if (!text || !text.trim()) return;

        try {
          LocalStorageManager.addOption({
            text: text.trim(),
            weight: 1,
            duration: null,
            tags: tags
          });

          const optionsList = document.getElementById('options-list');
          if (optionsList) {
            optionsList.innerHTML = renderOptionsList(LocalStorageManager.getOptions());
          }

          e.target.reset();
        } catch (error) {
          console.error('Failed to add option:', error);
        }
        return;
      }
    });
  }

  // Intercept HTMX requests for anonymous users
  document.addEventListener('htmx:configRequest', function(event) {
    // Skip if user is authenticated
    if (isAuthenticated()) return;

    // Get request details from event.detail
    const verb = event.detail.verb.toUpperCase();
    const path = event.detail.path;

    // POST /api/random - Select random option
    if (verb === 'POST' && path === '/api/random') {
      event.preventDefault();
      
      const formData = new FormData(event.detail.elt);
      const timeConstraint = parseTimeConstraint(formData);
      const tags = parseTags(formData);

      const selected = LocalStorageManager.selectRandom(timeConstraint, tags);
      // event.detail.target is the selector string (e.g., "#result")
      // We need to use querySelector to get the element
      const target = typeof event.detail.target === 'string' 
        ? document.querySelector(event.detail.target)
        : event.detail.target;
      
      if (target) {
        if (selected) {
          target.innerHTML = renderResult(selected);
        } else {
          target.innerHTML = renderNoOptions(timeConstraint || 0);
        }
      } else {
        console.error('[BRIDGE] Target element not found:', event.detail.target);
      }
      return;
    }

    // GET /manage/options - Show manage modal
    if (verb === 'GET' && path === '/manage/options') {
      event.preventDefault();
      
      const target = document.getElementById('manage-modal');
      if (target) {
        target.innerHTML = renderManageModal();
      }
      return;
    }

    // POST /api/options - Add new option
    if (verb === 'POST' && path === '/api/options') {
      event.preventDefault();
      
      const formData = new FormData(event.detail.elt);
      const text = formData.get('text');
      const tags = parseTags(formData);

      if (!text || !text.trim()) return;

      try {
        LocalStorageManager.addOption({
          text: text.trim(),
          weight: 1,
          duration: null,
          tags: tags
        });

        const target = document.getElementById('options-list');
        if (target) {
          target.innerHTML = renderOptionsList(LocalStorageManager.getOptions());
        }

        event.detail.elt.reset();
      } catch (error) {
        console.error('Failed to add option:', error);
      }
      return;
    }

    // DELETE /api/options/{id} - Delete option
    if (verb === 'DELETE' && path.startsWith('/api/options/')) {
      event.preventDefault();
      
      const optionId = path.split('/').pop();
      
      try {
        LocalStorageManager.deleteOption(optionId);
        
        const target = document.getElementById('options-list');
        if (target) {
          target.innerHTML = renderOptionsList(LocalStorageManager.getOptions());
        }
      } catch (error) {
        console.error('Failed to delete option:', error);
      }
      return;
    }

    // GET /close-modal - Close modal
    if (verb === 'GET' && path === '/close-modal') {
      event.preventDefault();
      const target = document.getElementById('manage-modal');
      if (target) {
        target.innerHTML = '';
      }
      return;
    }
  });

  // Listen for syncLocalStorage event (triggered on signin/signup)
  document.addEventListener('htmx:afterRequest', function(event) {
    if (!event.detail.successful) return;

    const triggerHeader = event.detail.xhr.getResponseHeader('HX-Trigger');
    if (!triggerHeader) return;

    try {
      const triggers = JSON.parse(triggerHeader);
      if (triggers.syncLocalStorage) {
        syncLocalStorageToServer();
      }
    } catch (e) {
      // Ignore parsing errors
    }
  });

  // Sync local storage options to server
  async function syncLocalStorageToServer() {
    const options = LocalStorageManager.getOptions();
    if (options.length === 0) return;

    try {
      const response = await fetch('/api/sync-local-options', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(options)
      });

      if (response.ok) {
        LocalStorageManager.clear();
        console.log('Local storage options synced to server');
      }
    } catch (error) {
      console.error('Failed to sync local storage:', error);
    }
  }

  // Initialize event delegation on page load
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', setupEventDelegation);
  } else {
    setupEventDelegation();
  }
})();
