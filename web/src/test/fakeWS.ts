export class FakeWS {
    static instances: FakeWS[] = []
    static readonly CONNECTING = 0
    static readonly OPEN = 1
    static readonly CLOSING = 2
    static readonly CLOSED = 3

    readyState = FakeWS.CONNECTING
    sent: string[] = []
    onmessage: ((e: { data: string }) => void) | null = null
    onclose: (() => void) | null = null
    onopen: (() => void) | null = null
    constructor(_url: string) {
        FakeWS.instances.push(this)
    }
    // open() models the socket handshake completing: real WebSockets throw
    // InvalidStateError when send() is called before this happens.
    open() {
        this.readyState = FakeWS.OPEN
        this.onopen?.()
    }
    send(d: string) {
        if (this.readyState !== FakeWS.OPEN) throw new Error('InvalidStateError: WebSocket is not open')
        this.sent.push(d)
    }
    close() {
        this.readyState = FakeWS.CLOSED
    }
}
