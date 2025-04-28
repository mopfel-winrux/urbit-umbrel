const prefix = import.meta.env.VITE_LAUNCH_PREFIX || ''
const toURL  = p => `${prefix}${p}`
const cfg = (m, body) => ({
  method: m,
  credentials: 'include',
  body: body || undefined
})

async function handle(resp) {
  if (resp.status === 401) {
    if (location.pathname !== '/login') location.replace('/login')
    throw new Error('unauth')
  }
  
  const ct = (resp.headers.get('content-type') || '').toLowerCase()
  const hasBody = resp.headers.get('content-length') !== '0'
  
  if (hasBody && ct.includes('application/json')) {
    return resp.json()
  }
  
  if (resp.status < 300) {
    return true
  }
  
  return null
}
  

export const login = pass =>
  fetch(toURL('/api/login'), {
    method: 'POST',
    credentials: 'include', 
    headers: {'content-type': 'application/json'},
    body: JSON.stringify({user: 'umbrel', pass})
  })
  .then(async r => {
    console.log('Login response status:', r.status);
    console.log('Login cookies:', document.cookie);
    
    if (r.status === 200) {
      localStorage.setItem('authenticated', 'true');
      return true;
    }
    return false;
  });



export const logout = () => fetch(toURL('/api/logout'), cfg('POST')).then(handle)
export const getStatus = () => 
  fetch(toURL('/api/status'), {
    method: 'GET',
    credentials: 'include',
    headers: {
      'Accept': 'application/json',
      'X-Requested-With': 'XMLHttpRequest'
    }
  })
  .then(handle);
export const stopUrbit = () => fetch(toURL('/api/stop'),   cfg('POST')).then(handle)
export const getLogs = () => fetch(toURL('/api/logs'), cfg('GET')).then(handle)

export const boot = (path,loom)=>
  fetch(toURL('/api/boot'), cfg('POST', JSON.stringify({path,loom}))).then(handle)

export const bootComet = loom=>
  fetch(toURL('/api/boot-comet'), cfg('POST', JSON.stringify({loom}))).then(handle)

export const uploadKey = file=>{
  const f=new FormData(); f.append('file',file)
  return fetch(toURL('/api/upload-key'), {...cfg('POST'), body:f}).then(handle)
}

export const uploadPier = fd =>
  fetch(toURL('/api/upload-pier'), {...cfg('POST'), body:fd}).then(handle)
