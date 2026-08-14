export class FakeWS {
  static instances: FakeWS[] = []
  sent: string[] = []
  onmessage: ((e: { data: string }) => void) | null = null
  constructor(_url: string) { FakeWS.instances.push(this) }
  send(d: string) { this.sent.push(d) }
  close() {}
}
