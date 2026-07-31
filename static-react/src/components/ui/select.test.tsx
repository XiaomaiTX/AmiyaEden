import userEvent from '@testing-library/user-event'
import { render, screen } from '@testing-library/react'
import { useState } from 'react'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

function SelectFixture({ disabled = false }: { disabled?: boolean }) {
  const [value, setValue] = useState('')

  return (
    <Select
      aria-label="Status"
      selectedKey={value}
      onSelectionChange={(key) => setValue(String(key))}
    >
      <SelectTrigger>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem id="">All</SelectItem>
        <SelectItem id="active">Active</SelectItem>
        <SelectItem id="disabled" isDisabled={disabled}>
          Disabled
        </SelectItem>
      </SelectContent>
    </Select>
  )
}

describe('Select', () => {
  test('round-trips an empty-string option through the React Aria value', async () => {
    const user = userEvent.setup()
    render(<SelectFixture />)

    await user.click(screen.getByRole('button', { name: /All Status/ }))
    await user.click(screen.getByRole('option', { name: 'Active' }))
    expect(screen.getByRole('button', { name: /Active Status/ })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Active Status/ }))
    await user.click(screen.getByRole('option', { name: 'All' }))
    expect(screen.getByRole('button', { name: /All Status/ })).toBeInTheDocument()
  })

  test('supports keyboard selection and disabled items', async () => {
    const user = userEvent.setup()
    render(<SelectFixture disabled />)

    await user.click(screen.getByRole('button', { name: /All Status/ }))
    expect(screen.getByRole('option', { name: 'Disabled' })).toHaveAttribute('data-disabled')
    await user.keyboard('{ArrowDown}{Enter}')

    expect(screen.getByRole('button', { name: /Active Status/ })).toBeInTheDocument()
  })
})
