import { TableCell, TableRow } from "@/components/ui/table";

interface EmptyTableRowProps {
  colSpan: number;
  message: string;
}

export function EmptyTableRow({ colSpan, message }: EmptyTableRowProps) {
  return (
    <TableRow>
      <TableCell colSpan={colSpan} className="text-center text-muted-foreground">
        {message}
      </TableCell>
    </TableRow>
  );
}
