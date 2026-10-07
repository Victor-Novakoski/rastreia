import { Component, type ReactNode } from 'react'

type Props = {
  /** O que aparece no lugar do que quebrou. */
  fallback: ReactNode
  children: ReactNode
}

/**
 * Um erro ao montar a tela (um pedaço do código que não baixou, por exemplo)
 * mostra o fallback no lugar daquele pedaço, em vez de apagar a página toda.
 */
export class ErrorBoundary extends Component<Props, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  render() {
    return this.state.failed ? this.props.fallback : this.props.children
  }
}
