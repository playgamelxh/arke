"""
Milvus Bridge - 轻量级 HTTP 网关
将 Milvus gRPC API 封装为 REST API，给 Go 后端调用
"""
import os
import time
import logging
from typing import Any, Dict, List, Optional

from flask import Flask, jsonify, request
from pymilvus import (
    connections,
    utility,
    Collection,
    CollectionSchema,
    FieldSchema,
    DataType,
)
from pymilvus.exceptions import MilvusException

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

MILVUS_HOST = os.getenv("MILVUS_HOST", "milvus")
MILVUS_PORT = int(os.getenv("MILVUS_PORT", "19530"))
PORT = int(os.getenv("PORT", "8088"))

app = Flask(__name__)


def get_connection() -> str:
    """获取或创建 Milvus 连接别名"""
    alias = "default"
    if alias not in connections.list_connections():
        connections.connect(alias=alias, host=MILVUS_HOST, port=MILVUS_PORT, timeout=10)
    return alias


@app.route("/health", methods=["GET"])
def health():
    """健康检查"""
    try:
        get_connection()
        return jsonify({"status": "ok", "milvus_address": f"{MILVUS_HOST}:{MILVUS_PORT}"})
    except Exception as e:
        return jsonify({"status": "error", "message": str(e)}), 500


@app.route("/collections", methods=["POST"])
def create_collection():
    """创建集合
    Body: {
        "collection_name": "kb_xxx",
        "dimension": 1024,
        "index_type": "HNSW",
        "index_params": {"M": 16, "efConstruction": 200},
        "metric_type": "COSINE"
    }
    """
    try:
        body = request.get_json(force=True)
        name = body["collection_name"]
        dim = body["dimension"]
        index_type = body.get("index_type", "HNSW")
        index_params = body.get("index_params", {})
        metric_type = body.get("metric_type", "COSINE")

        if utility.has_collection(name, using=get_connection()):
            return jsonify({"status": "exists", "collection_name": name})

        # 定义 schema
        fields = [
            FieldSchema(name="id", dtype=DataType.VARCHAR, max_length=64, is_primary=True),
            FieldSchema(name="kb_id", dtype=DataType.VARCHAR, max_length=64),
            FieldSchema(name="doc_id", dtype=DataType.VARCHAR, max_length=64),
            FieldSchema(name="segment_id", dtype=DataType.VARCHAR, max_length=64),
            FieldSchema(name="content", dtype=DataType.VARCHAR, max_length=8192),
            FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=dim),
        ]
        schema = CollectionSchema(fields=fields, description=f"Knowledge base {name}")
        coll = Collection(name=name, schema=schema, using=get_connection())

        # 创建索引
        coll.create_index(
            field_name="vector",
            index_params={"index_type": index_type, "metric_type": metric_type, "params": index_params},
        )

        return jsonify({"status": "created", "collection_name": name})
    except MilvusException as e:
        logger.exception("Create collection failed")
        return jsonify({"error": str(e)}), 400
    except Exception as e:
        logger.exception("Create collection failed")
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>", methods=["DELETE"])
def drop_collection(name: str):
    """删除集合"""
    try:
        utility.drop_collection(name, using=get_connection())
        return jsonify({"status": "dropped", "collection_name": name})
    except Exception as e:
        logger.exception("Drop collection failed")
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>/exists", methods=["GET"])
def collection_exists(name: str):
    """检查集合是否存在"""
    try:
        exists = utility.has_collection(name, using=get_connection())
        return jsonify({"collection_name": name, "exists": exists})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>/load", methods=["POST"])
def load_collection(name: str):
    """加载集合到内存（搜索前必须）"""
    try:
        coll = Collection(name, using=get_connection())
        coll.load()
        return jsonify({"status": "loaded", "collection_name": name})
    except Exception as e:
        logger.exception("Load collection failed")
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>/release", methods=["POST"])
def release_collection(name: str):
    """释放集合"""
    try:
        coll = Collection(name, using=get_connection())
        coll.release()
        return jsonify({"status": "released", "collection_name": name})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>/insert", methods=["POST"])
def insert(name: str):
    """插入向量
    Body: {
        "items": [
            {"id": "...", "kb_id": "...", "doc_id": "...", "segment_id": "...", "content": "...", "vector": [...]},
            ...
        ]
    }
    """
    try:
        body = request.get_json(force=True)
        items = body["items"]
        if not items:
            return jsonify({"status": "empty"})

        coll = Collection(name, using=get_connection())

        ids = [it["id"] for it in items]
        kb_ids = [it.get("kb_id", "") for it in items]
        doc_ids = [it.get("doc_id", "") for it in items]
        segment_ids = [it.get("segment_id", "") for it in items]
        contents = [it.get("content", "")[:8000] for it in items]
        vectors = [it["vector"] for it in items]

        coll.insert([ids, kb_ids, doc_ids, segment_ids, contents, vectors])
        coll.flush()

        return jsonify({"status": "inserted", "count": len(items)})
    except Exception as e:
        logger.exception("Insert failed")
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>/delete", methods=["POST"])
def delete_by_filter(name: str):
    """按 filter 删除
    Body: {"filter": "doc_id == 'xxx'"}
    """
    try:
        body = request.get_json(force=True)
        filter_expr = body["filter"]
        coll = Collection(name, using=get_connection())
        coll.load()
        coll.delete(filter_expr)
        coll.flush()
        return jsonify({"status": "deleted", "filter": filter_expr})
    except Exception as e:
        logger.exception("Delete failed")
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>/search", methods=["POST"])
def search(name: str):
    """向量检索
    Body: {
        "vectors": [[...], ...],
        "top_k": 5,
        "filter": "doc_id == 'xxx'" (可选),
        "output_fields": ["id", "content", "doc_id", "segment_id"]
    }
    """
    try:
        body = request.get_json(force=True)
        vectors = body["vectors"]
        top_k = body.get("top_k", 5)
        filter_expr = body.get("filter", "")
        output_fields = body.get("output_fields", ["id", "content", "doc_id", "segment_id"])

        coll = Collection(name, using=get_connection())
        coll.load()

        search_params = {"metric_type": "COSINE"}
        if filter_expr:
            results = coll.search(
                data=vectors,
                anns_field="vector",
                param=search_params,
                limit=top_k,
                expr=filter_expr,
                output_fields=output_fields,
            )
        else:
            results = coll.search(
                data=vectors,
                anns_field="vector",
                param=search_params,
                limit=top_k,
                output_fields=output_fields,
            )

        # 格式化结果
        formatted = []
        for hits in results:
            hit_list = []
            for hit in hits:
                item = {
                    "id": hit.id,
                    "distance": hit.distance,
                }
                for f in output_fields:
                    item[f] = hit.entity.get(f, "")
                hit_list.append(item)
            formatted.append(hit_list)

        return jsonify({"results": formatted})
    except Exception as e:
        logger.exception("Search failed")
        return jsonify({"error": str(e)}), 500


@app.route("/collections/<name>/count", methods=["GET"])
def count(name: str):
    """获取集合中的向量数量"""
    try:
        coll = Collection(name, using=get_connection())
        coll.flush()
        num = coll.num_entities
        return jsonify({"collection_name": name, "num_entities": num})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    logger.info(f"Starting Milvus Bridge on port {PORT}, connecting to {MILVUS_HOST}:{MILVUS_PORT}")
    # 启动时尝试连接
    for i in range(10):
        try:
            get_connection()
            logger.info("Connected to Milvus")
            break
        except Exception as e:
            logger.warning(f"Waiting for Milvus... ({i+1}/10): {e}")
            time.sleep(3)
    app.run(host="0.0.0.0", port=PORT)
