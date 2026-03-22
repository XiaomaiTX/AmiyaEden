const MILLION_ISK = 1_000_000
const INPUT_PRECISION = 2

function roundToInputPrecision(value: number) {
  return Number(value.toFixed(INPUT_PRECISION))
}

export function toMillionISKInput(value: number | null | undefined) {
  return roundToInputPrecision(Number(value ?? 0) / MILLION_ISK)
}

export function fromMillionISKInput(value: number | null | undefined) {
  return Math.round(roundToInputPrecision(Number(value ?? 0)) * MILLION_ISK)
}
