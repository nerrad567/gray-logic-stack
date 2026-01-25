"""Telegram history REST routes.

Provides access to the ring buffer of recently recorded telegrams
without requiring a WebSocket connection.
"""

from fastapi import APIRouter, Query

router = APIRouter(prefix="/api/v1/premises/{premise_id}/telegrams", tags=["telegrams"])


@router.get("")
def get_telegram_history(
    premise_id: str,
    limit: int = Query(default=100, ge=1, le=1000),
    offset: int = Query(default=0, ge=0),
):
    """Get recent telegram history for a premise (newest first)."""
    inspector = router.app.state.telegram_inspector
    entries = inspector.get_history(premise_id, limit=limit, offset=offset)
    stats = inspector.get_stats(premise_id)
    return {
        "telegrams": entries,
        "count": len(entries),
        "total_buffered": stats.get("buffered", 0),
    }


@router.get("/stats")
def get_telegram_stats(premise_id: str):
    """Get telegram statistics for a premise."""
    inspector = router.app.state.telegram_inspector
    return inspector.get_stats(premise_id)


@router.delete("")
def clear_telegram_history(premise_id: str):
    """Clear telegram history for a premise."""
    inspector = router.app.state.telegram_inspector
    inspector.clear(premise_id)
    return {"status": "cleared", "premise_id": premise_id}
