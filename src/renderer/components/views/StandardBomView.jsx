import React, { useMemo } from 'react'
import BomTable from '../tables/BomTable'

export default function StandardBomView({ data, isLoading, searchTerm, searchFields, viewContextIds }) {
    return (
        <BomTable
            data={data}
            isLoading={isLoading}
            searchTerm={searchTerm}
            searchFields={searchFields}
            mode="BOM"
            viewContextIds={viewContextIds}
        />
    )
}
