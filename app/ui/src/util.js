export const isAuthed = () => {
    // Log all cookies to see what's available
    console.log("All cookies in isAuthed:", document.cookie);
    
    // Try different cookie name patterns
    const hasUmbrelCookie = document.cookie.includes('umbrel=');
    const hasUmbrelUserCookie = document.cookie.includes('umbrel_user=');
    
    console.log("umbrel cookie present:", hasUmbrelCookie);
    console.log("umbrel_user cookie present:", hasUmbrelUserCookie);
    
    const hasLocalStorage = localStorage.getItem('authenticated') === 'true';
    console.log("localStorage auth present:", hasLocalStorage);
    
    return hasUmbrelCookie || hasUmbrelUserCookie || hasLocalStorage;
  }