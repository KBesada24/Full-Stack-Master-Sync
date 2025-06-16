"use client";

import { useState } from "react";

export default function StatefulPart() {
  const [apiUrl, setApiUrl] = useState("");
  
  return (
    <div>
      {/* Only the parts that need state go here */}
      <input 
        value={apiUrl}
        onChange={(e) => setApiUrl(e.target.value)}
      />
      <p>Current URL: {apiUrl}</p>
    </div>
  );
}
