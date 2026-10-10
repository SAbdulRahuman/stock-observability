"use client";

import { Float } from "@react-three/drei";
import { Canvas, useFrame, useThree } from "@react-three/fiber";
import { useEffect, useMemo, useRef, useState } from "react";
import * as THREE from "three";

export type Progress = { current: number };

const UP = new THREE.Color("#2fd47f");
const DOWN = new THREE.Color("#ff5a5f");
const PRIMARY = "#6f8cff";
const FOG = "#0f1220";
const SPACING = 0.55;

/** A market-like surface: slow trends, faster swings and tick noise. */
function height(x: number, z: number, t: number) {
  const trend = 0.5 + 0.5 * Math.sin(x * 0.42 + t * 0.7) * Math.cos(z * 0.5 - t * 0.45);
  const swing = 0.18 * Math.sin((x + z) * 1.05 + t * 1.9);
  const noise = 0.08 * Math.sin(x * 2.7 - t * 3.1 + z * 1.3);
  return Math.max(0.08, 0.25 + trend * 1.5 + swing + noise);
}

function TickField({ cols, rows, animate }: { cols: number; rows: number; animate: boolean }) {
  const mesh = useRef<THREE.InstancedMesh>(null);
  const invalidate = useThree((s) => s.invalidate);
  const count = cols * rows;
  const prev = useMemo(() => new Float32Array(count), [count]);
  const dummy = useMemo(() => new THREE.Object3D(), []);
  const color = useMemo(() => new THREE.Color(), []);

  const update = (t: number) => {
    const m = mesh.current;
    if (!m) return;
    let i = 0;
    for (let c = 0; c < cols; c++) {
      for (let r = 0; r < rows; r++, i++) {
        const x = (c - (cols - 1) / 2) * SPACING;
        const z = (r - (rows - 1) / 2) * SPACING;
        const h = height(x, z, t);
        dummy.position.set(x, h / 2, z);
        dummy.scale.set(1, h, 1);
        dummy.updateMatrix();
        m.setMatrixAt(i, dummy.matrix);
        color.copy(h >= prev[i] ? UP : DOWN).multiplyScalar(0.35 + Math.min(h, 2.4) * 0.28);
        m.setColorAt(i, color);
        prev[i] = h;
      }
    }
    m.instanceMatrix.needsUpdate = true;
    if (m.instanceColor) m.instanceColor.needsUpdate = true;
  };

  useEffect(() => {
    update(0);
    update(0.05);
    invalidate();
  }, [invalidate]);
  useFrame(({ clock }) => {
    if (animate) update(clock.elapsedTime);
  });

  return (
    <instancedMesh ref={mesh} args={[undefined, undefined, count]}>
      <boxGeometry args={[0.3, 1, 0.3]} />
      <meshStandardMaterial roughness={0.35} metalness={0.25} />
    </instancedMesh>
  );
}

/** The brand's eye: it turns towards the pointer. */
function Eye({ position }: { position: [number, number, number] }) {
  const group = useRef<THREE.Group>(null);
  useFrame(({ pointer }) => {
    const g = group.current;
    if (!g) return;
    g.rotation.y += (pointer.x * 0.5 - g.rotation.y) * 0.08;
    g.rotation.x += (-pointer.y * 0.3 - g.rotation.x) * 0.08;
  });
  return (
    <Float speed={1.3} floatIntensity={0.5} rotationIntensity={0.15}>
      <group ref={group} position={position}>
        <mesh>
          <torusGeometry args={[1.25, 0.055, 24, 160]} />
          <meshStandardMaterial color={PRIMARY} emissive={PRIMARY} emissiveIntensity={2.2} />
        </mesh>
        <mesh>
          <torusGeometry args={[0.88, 0.018, 16, 120]} />
          <meshStandardMaterial color="#a9b8ff" emissive="#a9b8ff" emissiveIntensity={1.2} />
        </mesh>
        <mesh>
          <sphereGeometry args={[0.44, 48, 48]} />
          <meshStandardMaterial color={FOG} emissive={PRIMARY} emissiveIntensity={0.55} roughness={0.15} metalness={0.7} />
        </mesh>
        <mesh position={[0, 0, 0.38]}>
          <sphereGeometry args={[0.13, 32, 32]} />
          <meshBasicMaterial color="#eef1ff" />
        </mesh>
      </group>
    </Float>
  );
}

/** Moves the camera with scroll progress and eases it towards the pointer. */
function Rig({ progress }: { progress: Progress }) {
  useFrame(({ camera, pointer }) => {
    const p = progress.current;
    camera.position.x += (pointer.x * 0.9 - camera.position.x) * 0.04;
    camera.position.y += (5.4 - p * 2.6 + pointer.y * 0.4 - camera.position.y) * 0.06;
    camera.position.z += (12.5 - p * 5 - camera.position.z) * 0.06;
    camera.lookAt(0, 1.4 + p * 0.6, 0);
  });
  return null;
}

export default function HeroScene({ progress, reduced, lite }: { progress: Progress; reduced: boolean; lite: boolean }) {
  const wrap = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(true);
  useEffect(() => {
    const el = wrap.current;
    if (!el) return;
    const io = new IntersectionObserver(([e]) => setVisible(e.isIntersecting));
    io.observe(el);
    return () => io.disconnect();
  }, []);
  return (
    <div ref={wrap} className="absolute inset-0" aria-hidden>
      <Canvas
        frameloop={reduced ? "demand" : visible ? "always" : "never"}
        dpr={[1, lite ? 1.5 : 2]}
        camera={{ position: [0, 5.4, 12.5], fov: 45 }}
        gl={{ antialias: !lite, alpha: true, powerPreference: "high-performance" }}
        onCreated={({ camera }) => camera.lookAt(0, 1.4, 0)}
      >
        <fog attach="fog" args={[FOG, 9, 24]} />
        <ambientLight intensity={0.55} />
        <directionalLight position={[6, 10, 5]} intensity={1.3} />
        <pointLight position={[0, 4, 1.5]} color={PRIMARY} intensity={28} distance={14} />
        <TickField cols={lite ? 30 : 48} rows={lite ? 15 : 24} animate={!reduced} />
        {/* Right of the left-aligned copy on desktop; the narrow mobile layout has no room for it. */}
        {!lite && <Eye position={[4.2, 3.2, 0]} />}
        {!reduced && <Rig progress={progress} />}
      </Canvas>
    </div>
  );
}
