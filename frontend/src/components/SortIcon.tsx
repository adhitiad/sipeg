import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'

type SortIconProps = {
  active: boolean
  order: 'asc' | 'desc'
}

export default function SortIcon({ active, order }: SortIconProps) {
  const Icon = !active ? ArrowUpDown : order === 'asc' ? ArrowUp : ArrowDown
  return <Icon className="ml-1 h-3.5 w-3.5" aria-hidden="true" />
}
