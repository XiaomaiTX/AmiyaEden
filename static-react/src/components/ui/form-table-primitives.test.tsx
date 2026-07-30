import { fireEvent, render, screen } from '@testing-library/react'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldDescription, FieldError, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

describe('shared form and table primitives', () => {
  test('renders semantic table structure', () => {
    render(
      <Table>
        <TableHeader><TableRow><TableHead>Name</TableHead></TableRow></TableHeader>
        <TableBody><TableRow><TableCell>Amiya</TableCell></TableRow></TableBody>
      </Table>
    )

    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'Amiya' })).toBeInTheDocument()
  })

  test('keeps form labels, errors, and checkbox state accessible', () => {
    render(
      <Field data-invalid>
        <FieldLabel htmlFor="title">Title</FieldLabel>
        <Input id="title" aria-invalid />
        <FieldDescription>Required for publication.</FieldDescription>
        <FieldError>Title is required.</FieldError>
        <Checkbox aria-label="Visible" />
      </Field>
    )

    expect(screen.getByLabelText('Title')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByText('Title is required.')).toBeInTheDocument()
    const checkbox = screen.getByRole('checkbox', { name: 'Visible' })
    fireEvent.click(checkbox)
    expect(checkbox).toHaveAttribute('data-state', 'checked')
  })
})
