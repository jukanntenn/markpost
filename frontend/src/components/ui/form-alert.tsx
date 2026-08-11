import { TriangleAlertIcon } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface FormAlertProps {
  message: string
}

export function FormAlert({ message }: FormAlertProps) {
  if (!message) return null
  return (
    <Alert variant="danger">
      <TriangleAlertIcon />
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  )
}
