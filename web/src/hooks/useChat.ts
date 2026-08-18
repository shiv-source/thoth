import { useCallback, useEffect } from 'react'
import {
    assistantDelta,
    assistantStart,
    assistantThinking,
    chatError,
    fetchTree,
    loadChat,
    resetChat,
    selectConversationId,
    selectLastTool,
    selectMessages,
    selectStreaming,
    selectThinking,
    selectThinkingText,
    stopStreaming,
    toolActivity,
    turnDone,
    userMessage
} from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import type { ChatMessage } from '../store/slices/chatSlice'
import { ChatSocket, type ServerMessage } from '../ws/chat'

export type { ChatMessage }

// useChat adapts the chat slice to the socket: WS frames become dispatches
// into the store's chat state machine, and send/cancel call into the wire.
// The conversation state lives in Redux, so it outlives the component.
export function useChat(socket: ChatSocket | null) {
    const dispatch = useAppDispatch()
    const messages = useAppSelector(selectMessages)
    const streaming = useAppSelector(selectStreaming)
    const conversationId = useAppSelector(selectConversationId)
    const lastTool = useAppSelector(selectLastTool)
    const thinking = useAppSelector(selectThinking)
    const thinkingText = useAppSelector(selectThinkingText)

    const send = useCallback(
        (text: string) => {
            dispatch(userMessage(text))
            socket?.send(text)
        },
        [dispatch, socket]
    )

    const cancel = useCallback(() => {
        socket?.cancel()
        dispatch(stopStreaming())
    }, [dispatch, socket])

    const handle = useCallback(
        (m: ServerMessage) => {
            switch (m.type) {
                case 'assistant_start':
                    dispatch(assistantStart())
                    break
                case 'assistant_thinking':
                    dispatch(assistantThinking(m.text))
                    break
                case 'assistant_delta':
                    dispatch(assistantDelta(m.text))
                    break
                case 'tool_activity':
                    dispatch(toolActivity(toolLabel(m.tool, m.detail)))
                    break
                case 'turn_done':
                    // The server sends the conversation id on every finished turn; keep
                    // it so a reconnect can resume this conversation.
                    dispatch(turnDone(m.conversation_id ?? null))
                    break
                case 'wiki_changed':
                    // The watcher saw wiki files change: the tree is stale,
                    // refetch it instead of polling on every turn.
                    void dispatch(fetchTree())
                    break
                case 'error':
                    // Surface cancelled/crash feedback as a visible assistant message so
                    // the user knows the turn did not complete.
                    dispatch(chatError(m.message))
                    break
            }
        },
        [dispatch]
    )

    // load replaces the whole conversation with history fetched from the server.
    // Local only — the caller pins the server side via socket.open(conversationId).
    const load = useCallback(
        (msgs: ChatMessage[], convId: string) => {
            dispatch(loadChat({ messages: msgs, conversationId: convId }))
        },
        [dispatch]
    )

    const reset = useCallback(() => {
        // Unpin the server too: without the frame the server keeps the old
        // pinned conversation and the next send would continue it (interfering
        // chats into each other's history).
        socket?.newChat()
        dispatch(resetChat())
    }, [dispatch, socket])

    useEffect(() => {
        if (socket) socket.onMessage(handle)
    }, [socket, handle])

    return { messages, streaming, conversationId, lastTool, thinking, thinkingText, send, cancel, load, reset }
}

/** Pick the label for the tool status line: a path from the input JSON when
 *  present, otherwise the tool name itself. */
function toolLabel(tool: string, detail: string): string {
    try {
        const input = JSON.parse(detail) as { path?: unknown }
        if (typeof input.path === 'string' && input.path) return input.path
    } catch {
        // detail is not JSON — fall through to the tool name
    }
    return tool
}
