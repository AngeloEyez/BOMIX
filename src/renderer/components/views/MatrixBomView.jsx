import React from 'react'
import BomTable from '../tables/BomTable'

export default function MatrixBomView({ data, isLoading, searchTerm, searchFields, viewContextIds }) {
    return (
        <BomTable
            data={data}
            isLoading={isLoading}
            searchTerm={searchTerm}
            searchFields={searchFields}
            mode="MATRIX"
            viewContextIds={viewContextIds}
        />
    )
}
