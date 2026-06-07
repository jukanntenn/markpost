"use client";

import { PageHeading } from "@/components/ui/page-heading";
import { QueryState } from "@/components/ui/query-state";
import {
  Table,
  TableBody,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { EmptyTableRow } from "@/components/ui/empty-table-row";
import type { QueryStateProps } from "@/types/query-state";

interface AdminTablePageProps<T> extends QueryStateProps {
  title: string;
  toolbar?: React.ReactNode;
  emptyText: string;
  headers: React.ReactNode;
  colSpan: number;
  items: T[];
  renderRow: (item: T) => React.ReactNode;
}

export function AdminTablePage<T>({
  title,
  toolbar,
  isLoading,
  error,
  loadingText,
  errorText,
  emptyText,
  headers,
  colSpan,
  items,
  renderRow,
}: AdminTablePageProps<T>) {
  return (
    <div>
      <PageHeading actions={toolbar}>{title}</PageHeading>

      <QueryState isLoading={isLoading} error={error} loadingText={loadingText} errorText={errorText}>
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>{headers}</TableRow>
            </TableHeader>
            <TableBody>
              {items.length === 0 ? (
                <EmptyTableRow colSpan={colSpan} message={emptyText} />
              ) : (
                items.map(renderRow)
              )}
            </TableBody>
          </Table>
        </div>
      </QueryState>
    </div>
  );
}
