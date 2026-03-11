import * as React from "react"
import { AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"

interface Props {
  children: React.ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("Uncaught error:", error, errorInfo)
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center min-h-[50vh] p-8 text-center bg-background text-foreground rounded-lg border border-border shadow-sm m-4">
          <AlertCircle className="w-12 h-12 text-destructive mb-4" />
          <h2 className="text-2xl font-bold mb-2">Something went wrong</h2>
          <div className="bg-muted p-4 rounded mb-6 max-w-2xl overflow-auto text-left w-full">
            <p className="font-mono text-sm text-muted-foreground whitespace-pre-wrap">
              {this.state.error?.message || "An unknown error occurred"}
            </p>
          </div>
          <Button onClick={this.handleRetry} variant="default">
            Try again
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}
