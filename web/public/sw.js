// Service worker do Rastreia: mostra as notificações de Web Push que o
// worker manda (internal/notify/push.go) e abre o rastreio no toque.
self.addEventListener('push', (event) => {
  let data = {}
  try {
    data = event.data ? event.data.json() : {}
  } catch {
    // Sem JSON, mostra um aviso genérico.
  }
  const title = data.title || 'Rastreia'
  event.waitUntil(
    self.registration.showNotification(title, {
      body: data.body || 'Sua entrega foi atualizada.',
      icon: '/icon-192.png',
      badge: '/icon-192.png',
      tag: data.tag,
      renotify: Boolean(data.tag),
      data: { url: data.url || '/rastreio' },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  // Só abre páginas do próprio site, mesmo que a mensagem traga outra URL.
  const url = new URL(event.notification.data?.url || '/rastreio', self.location.origin)
  const target = url.origin === self.location.origin ? url.href : self.location.origin + '/rastreio'
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windows) => {
      for (const w of windows) {
        if (w.url === target && 'focus' in w) return w.focus()
      }
      return self.clients.openWindow(target)
    }),
  )
})
