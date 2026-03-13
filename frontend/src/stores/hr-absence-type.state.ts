import { defineStore } from 'pinia';

import {
  absenceTypeService,
  type AbsenceType,
  type ListAbsenceTypesResponse,
} from '../api/client';

export const useHrAbsenceTypeStore = defineStore('hr-absence-type', () => {
  async function listAbsenceTypes(
    paging?: { page?: number; pageSize?: number },
    formValues?: { query?: string } | null,
  ): Promise<ListAbsenceTypesResponse> {
    return await absenceTypeService.ListAbsenceTypes({
      query: formValues?.query,
      page: paging?.page,
      pageSize: paging?.pageSize,
    });
  }

  async function getAbsenceType(id: string) {
    return await absenceTypeService.GetAbsenceType({ id });
  }

  async function createAbsenceType(data: Partial<AbsenceType>) {
    return await absenceTypeService.CreateAbsenceType(data as any);
  }

  async function updateAbsenceType(
    id: string,
    data: Partial<AbsenceType>,
    updateMask: string[],
  ) {
    return await absenceTypeService.UpdateAbsenceType({
      id,
      data: data as AbsenceType,
      updateMask: updateMask.join(','),
    });
  }

  async function deleteAbsenceType(id: string) {
    return await absenceTypeService.DeleteAbsenceType({ id });
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
