const logPre = document.querySelector("#log")

const log = msg =>
  (logPre.textContent += `${new Date().toLocaleTimeString()} ${msg}\n`)

const socket = new WebSocket(
  `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/server`,
)

const eventTypes = ["close", "error", "open"]
for (let i = 0; i < eventTypes.length; i++) {
  const eventType = eventTypes[i]
  socket.addEventListener(eventType, e => log(eventType))
}

const onTimeout = () => {
  log("timed out waiting for message")
  socket.close()
}

let readTimeout
let timer

socket.addEventListener("message", e => {
  const data = JSON.parse(e.data)
  if (data.read_timeout) {
    readTimeout = data.read_timeout * 1000
  } else if (data.ping != undefined) {
    log(`received ping, sending pong ${data.ping}`)
    socket.send(JSON.stringify({ pong: data.ping }))
  }
  clearTimeout(timer)
  timer = setTimeout(onTimeout, readTimeout)
})
