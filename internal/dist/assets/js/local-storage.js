// LocalStorageManager - Manages wheel options in browser local storage for anonymous users
// TTL: 7 days
const LocalStorageManager = (function() {
  const STORAGE_KEY = 'wheel_options';
  const EXPIRY_KEY = 'wheel_options_expiry';
  const TTL_DAYS = 7;
  const TTL_MS = TTL_DAYS * 24 * 60 * 60 * 1000;

  // Initialize on page load - check expiry
  function init() {
    if (isExpired()) {
      clear();
    } else {
      updateExpiry();
    }
  }

  // Check if storage has expired
  function isExpired() {
    const expiry = localStorage.getItem(EXPIRY_KEY);
    if (!expiry) return false;
    return Date.now() > parseInt(expiry, 10);
  }

  // Update expiry timestamp (extends TTL on each interaction)
  function updateExpiry() {
    const expiry = Date.now() + TTL_MS;
    localStorage.setItem(EXPIRY_KEY, expiry.toString());
  }

  // Get all options from storage
  function getOptions() {
    try {
      const data = localStorage.getItem(STORAGE_KEY);
      if (!data) return [];
      const options = JSON.parse(data);
      return Array.isArray(options) ? options : [];
    } catch (e) {
      console.error('Failed to parse options from localStorage:', e);
      return [];
    }
  }

  // Save options to storage
  function saveOptions(options) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(options));
      updateExpiry();
    } catch (e) {
      console.error('Failed to save options to localStorage:', e);
    }
  }

  // Generate unique ID for local options
  function generateID() {
    const timestamp = Date.now();
    const random = Math.floor(Math.random() * 10000);
    return `local-${timestamp}-${random}`;
  }

  // Validate and clamp values
  function validateOption(option) {
    return {
      id: option.id || generateID(),
      text: (option.text || '').trim(),
      weight: Math.max(1, Math.min(10, parseInt(option.weight, 10) || 1)),
      duration: option.duration === null || option.duration === undefined ? null : Math.max(0, Math.min(1440, parseInt(option.duration, 10))),
      tags: Array.isArray(option.tags) ? option.tags.slice(0, 5).map(t => t.trim().toLowerCase()) : []
    };
  }

  // Add new option
  function addOption(option) {
    const options = getOptions();
    const validated = validateOption(option);
    
    if (!validated.text) {
      throw new Error('Option text is required');
    }

    options.push(validated);
    saveOptions(options);
    return validated;
  }

  // Get option by ID
  function getOption(id) {
    const options = getOptions();
    return options.find(opt => opt.id === id);
  }

  // Update option
  function updateOption(id, updates) {
    const options = getOptions();
    const index = options.findIndex(opt => opt.id === id);
    
    if (index === -1) {
      throw new Error('Option not found');
    }

    const updated = validateOption({
      ...options[index],
      ...updates,
      id: options[index].id // preserve original ID
    });

    options[index] = updated;
    saveOptions(options);
    return updated;
  }

  // Delete option
  function deleteOption(id) {
    const options = getOptions();
    const filtered = options.filter(opt => opt.id !== id);
    
    if (filtered.length === options.length) {
      throw new Error('Option not found');
    }

    saveOptions(filtered);
    return true;
  }

  // Clear all options
  function clear() {
    localStorage.removeItem(STORAGE_KEY);
    localStorage.removeItem(EXPIRY_KEY);
  }

  // Get all unique tags
  function getAllTags() {
    const options = getOptions();
    const tagSet = new Set();
    
    options.forEach(opt => {
      if (Array.isArray(opt.tags)) {
        opt.tags.forEach(tag => tagSet.add(tag));
      }
    });

    return Array.from(tagSet).sort();
  }

  // Get total weight (for weighted random selection)
  function getTotalWeight(options) {
    return options.reduce((sum, opt) => sum + opt.weight, 0);
  }

  // Filter options by time constraint and tags
  function filterOptions(timeConstraint, tags) {
    let options = getOptions();

    // Filter by time constraint
    if (timeConstraint && timeConstraint !== 'any') {
      const minutes = parseInt(timeConstraint, 10);
      options = options.filter(opt => {
        if (opt.duration === null || opt.duration === undefined) return true;
        return opt.duration <= minutes;
      });
    }

    // Filter by tags (must have ALL specified tags)
    if (tags && tags.length > 0) {
      options = options.filter(opt => {
        if (!Array.isArray(opt.tags)) return false;
        return tags.every(tag => opt.tags.includes(tag.toLowerCase()));
      });
    }

    return options;
  }

  // Select random option using weighted selection
  function selectRandom(timeConstraint, tags) {
    const options = filterOptions(timeConstraint, tags);

    if (options.length === 0) {
      return null;
    }

    const totalWeight = getTotalWeight(options);
    let random = Math.random() * totalWeight;

    for (const option of options) {
      random -= option.weight;
      if (random <= 0) {
        return option;
      }
    }

    // Fallback (should never reach here)
    return options[options.length - 1];
  }

  // Get options count
  function getCount() {
    return getOptions().length;
  }

  // Export all functions
  return {
    init,
    isExpired,
    updateExpiry,
    getOptions,
    addOption,
    getOption,
    updateOption,
    deleteOption,
    clear,
    getAllTags,
    getTotalWeight,
    filterOptions,
    selectRandom,
    getCount
  };
})();

// Auto-initialize on page load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => LocalStorageManager.init());
} else {
  LocalStorageManager.init();
}
