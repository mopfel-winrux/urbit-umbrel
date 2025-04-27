export const isAuthed = () => document.cookie.split('; ').some(c => c.startsWith('umbrel=') && c.split('=')[1])
