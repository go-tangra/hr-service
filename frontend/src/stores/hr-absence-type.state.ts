import { defineStore } from 'pinia';

import {
  AbsenceTypeService,
  type AbsenceType,
  type ListAbsenceTypesResponse,
} from '../api/services';
import type { Paging } from '../api/services';

export const useHrAbsenceTypeStore = defineStore('hr-absence-type', () => {
  async function listAbsenceTypes(
    paging?: Paging,
    formValues?: {
      query?: string;
    } | null,
  ): Promise<ListAbsenceTypesResponse> {
    return await AbsenceTypeService.list({
      query: formValues?.query,
      page: paging?.page,
      pageSize: paging?.pageSize,
    });
  }

  async function getAbsenceType(
    id: string,
  ): Promise<{ absenceType: AbsenceType }> {
    return await AbsenceTypeService.get(id);
  }

  async function createAbsenceType(
    data: Partial<AbsenceType>,
  ): Promise<{ absenceType: AbsenceType }> {
    return await AbsenceTypeService.create(data);
  }

  async function updateAbsenceType(
    id: string,
    data: Partial<AbsenceType>,
    updateMask: string[],
  ): Promise<{ absenceType: AbsenceType }> {
    return await AbsenceTypeService.update(id, {
      id,
      data: data as AbsenceType,
      updateMask: updateMask.join(','),
    });
  }

  async function deleteAbsenceType(id: string): Promise<void> {
    return await AbsenceTypeService.delete(id);
  }

  function $reset() {}

  return {
    $reset,
    listAbsenceTypes,
    getAbsenceType,
    createAbsenceType,
    updateAbsenceType,
    deleteAbsenceType,
  };
});
