# WS Test Doubles

> 21 nodes · cohesion 0.11

## Key Concepts

- **ChatSocket** (20 connections) — `web/src/ws/chat.ts`
- **FakeWS** (8 connections) — `web/src/test/fakeWS.ts`
- **.connect()** (6 connections) — `web/src/ws/chat.ts`
- **fakeWS.ts** (4 connections) — `web/src/test/fakeWS.ts`
- **chat.test.ts** (4 connections) — `web/src/ws/chat.test.ts`
- **freshSocket()** (2 connections) — `web/src/hooks/useChat.test.tsx`
- **.handler()** (2 connections) — `web/src/ws/chat.ts`
- **.resume()** (2 connections) — `web/src/ws/chat.ts`
- **.statusHandler()** (2 connections) — `web/src/ws/chat.ts`
- **.close()** (1 connections) — `web/src/test/fakeWS.ts`
- **.constructor()** (1 connections) — `web/src/test/fakeWS.ts`
- **.open()** (1 connections) — `web/src/test/fakeWS.ts`
- **.send()** (1 connections) — `web/src/test/fakeWS.ts`
- **.cancel()** (1 connections) — `web/src/ws/chat.ts`
- **.close()** (1 connections) — `web/src/ws/chat.ts`
- **.constructor()** (1 connections) — `web/src/ws/chat.ts`
- **.newChat()** (1 connections) — `web/src/ws/chat.ts`
- **.onMessage()** (1 connections) — `web/src/ws/chat.ts`
- **.onStatusChange()** (1 connections) — `web/src/ws/chat.ts`
- **.open()** (1 connections) — `web/src/ws/chat.ts`
- **.send()** (1 connections) — `web/src/ws/chat.ts`

## Relationships

- [Chat Hook](Chat_Hook.md) (7 shared connections)
- [Conversation Routing](Conversation_Routing.md) (3 shared connections)
- [Component Tests](Component_Tests.md) (2 shared connections)
- [Chat Panel](Chat_Panel.md) (2 shared connections)

## Source Files

- `web/src/hooks/useChat.test.tsx`
- `web/src/test/fakeWS.ts`
- `web/src/ws/chat.test.ts`
- `web/src/ws/chat.ts`

## Audit Trail

- EXTRACTED: 36 (95%)
- INFERRED: 2 (5%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*