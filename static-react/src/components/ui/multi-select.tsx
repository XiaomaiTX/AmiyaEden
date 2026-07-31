import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
} from '@/components/ui/combobox'
import { Button } from '@/components/ui/button'
import { XIcon } from 'lucide-react'

export type MultiSelectOption = {
  value: string
  label: string
  disabled?: boolean
}

type MultiSelectProps = {
  value: string[]
  onValueChange: (value: string[]) => void
  options: readonly MultiSelectOption[]
  placeholder?: string
  emptyText?: string
  disabled?: boolean
  className?: string
  onInputValueChange?: (value: string) => void
}

function MultiSelect({
  value,
  onValueChange,
  options,
  placeholder,
  emptyText,
  disabled = false,
  className,
  onInputValueChange,
}: MultiSelectProps) {
  return (
    <Combobox<MultiSelectOption>
      selectedKey={null}
      onSelectionChange={(key) => {
        if (key === null) return
        const nextValue = String(key)
        onValueChange(
          value.includes(nextValue)
            ? value.filter((item) => item !== nextValue)
            : [...value, nextValue]
        )
      }}
      onInputChange={onInputValueChange}
      isDisabled={disabled}
    >
      <ComboboxChips className={className}>
        <ComboboxValue>
          {() => (
            <>
              {value.map((item) => (
                <ComboboxChip key={item} id={item} showRemove={false}>
                  {options.find((option) => option.value === item)?.label ?? item}
                  <Button
                    type="button"
                    size="icon-xs"
                    variant="ghost"
                    onPress={() => onValueChange(value.filter((selected) => selected !== item))}
                  >
                    <XIcon />
                  </Button>
                </ComboboxChip>
              ))}
              <ComboboxChipsInput
                placeholder={value.length ? '' : placeholder}
                aria-label={placeholder}
              />
            </>
          )}
        </ComboboxValue>
      </ComboboxChips>
      <ComboboxContent>
        {emptyText ? <ComboboxEmpty>{emptyText}</ComboboxEmpty> : null}
        <ComboboxList>
          {options.map((option) => (
            <ComboboxItem key={option.value} id={option.value} value={option} isDisabled={option.disabled}>
              {option.label}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}

export { MultiSelect }
