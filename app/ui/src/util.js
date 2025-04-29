export const isAuthed = () => {
    const hasUmbrelCookie = document.cookie.includes('umbrel=');
    const hasUmbrelUserCookie = document.cookie.includes('umbrel_user=');
    const hasLocalStorage = localStorage.getItem('authenticated') === 'true';
    return hasUmbrelCookie || hasUmbrelUserCookie || hasLocalStorage;
  }