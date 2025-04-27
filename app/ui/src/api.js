const prefix = import.meta.env.VITE_LAUNCH_PREFIX || ''
const toURL  = p => `${prefix}${p}`
const cfg    = (m, body)=>({
  method: m,
  credentials: 'include',
  headers: body ? {'content-type':'application/json'} : undefined,
  body
})

async function handle (resp) {
    if (resp.status === 401) {
      if (location.pathname !== '/login') location.replace('/login')
      throw new Error('unauth')
    }
    if ([201,202,204].includes(resp.status)) return null
    const ct = (resp.headers.get('content-type') || '').toLowerCase()
    const hasBody = resp.headers.get('content-length') !== '0'
    if (hasBody && ct.includes('application/json')) return resp.json()
    return null
}
  

export const login = pass =>
  fetch(toURL('/api/login'), cfg('POST', JSON.stringify({user:'umbrel',pass})))
  .then(r => r.ok)

export const logout = () => fetch(toURL('/api/logout'), cfg('POST')).then(handle)
export const getStatus = () => fetch(toURL('/api/status'), cfg('GET')).then(handle)
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
