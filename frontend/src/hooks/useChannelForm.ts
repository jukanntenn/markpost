import { useState } from "react";
import type { UseMutationResult } from "@tanstack/react-query";
import type { DeliveryChannel, DeliveryChannelResponse, CreateChannelPayload } from "@/types/delivery";
import type { FormState, UpdateChannelMutationVars } from "@/utils/channel-form";
import {
  EMPTY_FORM,
  channelToForm,
  formToCreatePayload,
  formToUpdatePayload,
} from "@/utils/channel-form";

export type { FormState } from "@/utils/channel-form";
export type { UpdateChannelMutationVars } from "@/utils/channel-form";

interface UseChannelFormOptions {
  createMutation: UseMutationResult<DeliveryChannelResponse, Error, CreateChannelPayload>;
  updateMutation: UseMutationResult<DeliveryChannelResponse, Error, UpdateChannelMutationVars>;
}

interface UseChannelFormReturn {
  form: FormState;
  setForm: (form: FormState) => void;
  showForm: boolean;
  editingId: number | null;
  isSubmitting: boolean;
  resetForm: () => void;
  startEdit: (channel: DeliveryChannel) => void;
  handleSubmit: (e: React.FormEvent) => void;
  openNewForm: () => void;
}

export function useChannelForm({
  createMutation,
  updateMutation,
}: UseChannelFormOptions): UseChannelFormReturn {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState(EMPTY_FORM);

  function resetAllFormState() {
    setEditingId(null);
    setForm(EMPTY_FORM);
  }

  function resetForm() {
    resetAllFormState();
    setShowForm(false);
  }

  function startEdit(channel: DeliveryChannel) {
    setEditingId(channel.id);
    setForm(channelToForm(channel));
    setShowForm(true);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (editingId !== null) {
      updateMutation.mutate(formToUpdatePayload(editingId, form));
    } else {
      createMutation.mutate(formToCreatePayload(form));
    }
  }

  function openNewForm() {
    resetAllFormState();
    setShowForm(true);
  }

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  return {
    form,
    setForm,
    showForm,
    editingId,
    isSubmitting,
    resetForm,
    startEdit,
    handleSubmit,
    openNewForm,
  };
}
