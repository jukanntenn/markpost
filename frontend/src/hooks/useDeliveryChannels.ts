"use client";

import { QueryClient, useQuery, useQueryClient } from "@tanstack/react-query";
import { deliveryApi, deliveryKeys, invalidateKey } from "@/lib/api";

export function invalidateDeliveryChannels(queryClient: QueryClient) {
  return invalidateKey(queryClient, deliveryKeys.channels());
}

export function useDeliveryChannels() {
  const queryClient = useQueryClient();

  const { data, ...rest } = useQuery({
    queryKey: deliveryKeys.channels(),
    queryFn: deliveryApi.list,
  });

  return {
    channels: data?.items || [],
    invalidate: () => invalidateDeliveryChannels(queryClient),
    ...rest,
  };
}
