import React, { useEffect } from 'react';
import useMatrixStore from '../../stores/useMatrixStore';
import { Trash2, Plus, RefreshCw, X } from 'lucide-react';

const MatrixModelDialog = ({ isOpen, onClose, bomRevisionId }) => {
    const { matrixData, fetchMatrixData, createModels, updateModel, deleteModel, isLoading, error } = useMatrixStore();
    const models = matrixData[bomRevisionId]?.models || [];

    // Local state for tracking changes before update (optional, but good for UX)
    // Actually, let's do direct updates for simplicity first, maybe with debounce if needed.
    // For now, "onBlur" update is good.

    useEffect(() => {
        if (isOpen && bomRevisionId) {
            fetchMatrixData(bomRevisionId);
        }
    }, [isOpen, bomRevisionId, fetchMatrixData]);

    const handleCreateDefault = async () => {
        await createModels(bomRevisionId); // Defaults
    };

    const handleAddModel = async () => {
        await createModels(bomRevisionId, [{ name: 'New Model', description: '' }]);
    };

    const handleUpdate = async (id, field, value) => {
        await updateModel(bomRevisionId, id, { [field]: value });
    };

    const handleDelete = async (id, name) => {
        if (confirm(`確定要刪除 Model "${name}" 嗎？\n如果有相關聯的選擇紀錄，必須先清除選擇才能刪除。`)) {
            await deleteModel(bomRevisionId, id);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-2xl overflow-hidden flex flex-col max-h-[90vh]">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
                    <h2 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
                        Matrix Model 管理
                    </h2>
                    <button
                        onClick={onClose}
                        className="p-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                    >
                        <X className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 overflow-y-auto flex-1">
                    {error && (
                        <div className="mb-4 p-3 bg-red-100 border border-red-200 text-red-700 rounded-lg text-sm">
                            {error}
                        </div>
                    )}

                    {isLoading && (
                        <div className="flex justify-center p-4">
                            <RefreshCw className="w-6 h-6 animate-spin text-primary-500" />
                        </div>
                    )}

                    {!isLoading && models.length === 0 && (
                        <div className="text-center py-8">
                            <p className="text-gray-500 dark:text-gray-400 mb-4">
                                目前沒有任何 Matrix Model
                            </p>
                            <button
                                onClick={handleCreateDefault}
                                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors flex items-center gap-2 mx-auto"
                            >
                                <RefreshCw className="w-4 h-4" />
                                初始化預設 Models (A, B, C)
                            </button>
                        </div>
                    )}

                    {models.length > 0 && (
                        <div className="space-y-3">
                            {models.map((model) => (
                                <div key={model.id} className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg border border-gray-200 dark:border-gray-700">
                                    <div className="flex-1 space-y-2">
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm font-medium text-gray-500 w-12">名稱</span>
                                            <input
                                                type="text"
                                                defaultValue={model.name}
                                                onBlur={(e) => {
                                                    if (e.target.value !== model.name) {
                                                        handleUpdate(model.id, 'name', e.target.value);
                                                    }
                                                }}
                                                className="flex-1 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded px-2 py-1 text-sm focus:ring-2 focus:ring-primary-500 outline-none"
                                            />
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm font-medium text-gray-500 w-12">描述</span>
                                            <input
                                                type="text"
                                                defaultValue={model.description}
                                                onBlur={(e) => {
                                                    if (e.target.value !== model.description) {
                                                        handleUpdate(model.id, 'description', e.target.value);
                                                    }
                                                }}
                                                className="flex-1 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded px-2 py-1 text-sm focus:ring-2 focus:ring-primary-500 outline-none"
                                                placeholder="選填..."
                                            />
                                        </div>
                                    </div>
                                    <button
                                        onClick={() => handleDelete(model.id, model.name)}
                                        className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                                        title="刪除"
                                    >
                                        <Trash2 className="w-5 h-5" />
                                    </button>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 flex justify-between items-center">
                    <span className="text-sm text-gray-500">
                        共 {models.length} 個 Model
                    </span>
                    <button
                        onClick={handleAddModel}
                        className="px-4 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-600 rounded-lg transition-colors flex items-center gap-2 text-sm font-medium"
                    >
                        <Plus className="w-4 h-4" />
                        新增 Model
                    </button>
                </div>
            </div>
        </div>
    );
};

export default MatrixModelDialog;
