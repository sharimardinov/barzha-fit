import { useState, useCallback } from "react";
import { api } from "../services/api";

export function useApi() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const call = useCallback(async (path, body) => {
    setLoading(true);
    setError(null);
    try {
      const data = await api(path, body);
      return data;
    } catch (err) {
      setError(err);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { call, loading, error };
}
