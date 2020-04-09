const READ_TIMEOUT = 10 * 1000

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
  socket.close()
  log("timed out waiting for message")
}

let timer = setTimeout(onTimeout, READ_TIMEOUT)

socket.addEventListener("message", e => {
  log(`received ping, sending pong ${e.data}`)
  socket.send(e.data) // pong
  clearTimeout(timer)
  timer = setTimeout(onTimeout, READ_TIMEOUT)
})
