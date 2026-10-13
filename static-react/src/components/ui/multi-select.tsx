import {
  Combobox,
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
  removeLabel?: (label: string) => string
  ariaLabel?: string
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
  removeLabel,
  ariaLabel,
  disabled = false,
  className,
  onInputValueChange,
}: MultiSelectProps) {
  return (
    <Combobox<MultiSelectOption>
      aria-label={ariaLabel ?? placeholder}
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
                <span
                  key={item}
                  className="flex h-[calc(--spacing(5.25))] w-fit items-center gap-1 rounded-sm bg-muted px-1.5 text-xs font-medium text-foreground"
                >
                  {options.find((option) => option.value === item)?.label ?? item}
                  <Button
                    type="button"
                    size="icon-xs"
                    variant="ghost"
                    aria-label={
                      removeLabel?.(
                        options.find((option) => option.value === item)?.label ?? item
                      ) ?? item
                    }
                    onPress={() => onValueChange(value.filter((selected) => selected !== item))}
                  >
                    <XIcon />
                  </Button>
                </span>
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
            <ComboboxItem
              key={option.value}
              id={option.value}
              value={option}
              isDisabled={option.disabled}
            >
              {option.label}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}

export { MultiSelect }
